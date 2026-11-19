package note

import (
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/agonzaNET/memctl/internal/index"
	"github.com/agonzaNET/memctl/internal/parse"
)

// MoveOp is the input of Move (note_move): at least one of the two.
type MoveOp struct{ NewPath, NewTitle string }

// Moved is the result of a successful Move.
type Moved struct {
	OldPath, Path, Title, SHA256 string
	Bytes                        int
	Linkers                      int  // other notes that link to the note
	Same                         bool // the note already had that path and title; nothing was written
}

// Move moves or renames a note (SPEC-0007 R20–R23), as one tool write. It
// never changes another note: the old path and the old title go into
// aliases, so links to them keep resolving (design D24).
func (s *Service) Move(resolve Resolver, op MoveOp) (Moved, error) {
	if op.NewPath == "" && op.NewTitle == "" {
		return Moved{}, &ArgError{Arg: "new_path", Reason: "or new_title is required"}
	}
	if op.NewTitle != "" {
		if err := checkLine("new_title", op.NewTitle, MaxTitle); err != nil {
			return Moved{}, err
		}
	}
	var out Moved
	err := s.mutate(func(idx *index.Index) (Change, error) {
		rel, err := resolve(idx)
		if err != nil {
			return Change{}, err
		}
		n, _, _ := idx.Get(rel)
		oldAbs, err := s.SafePath(rel)
		if err != nil {
			return Change{}, err
		}

		title := n.Title
		if op.NewTitle != "" {
			title = op.NewTitle
		}
		newRel := op.NewPath
		if newRel == "" {
			name, err := FileName(op.NewTitle)
			if err != nil {
				return Change{}, &ArgError{Arg: "new_title", Reason: "has no characters that can form a file name"}
			}
			newRel = name
			if dir := path.Dir(rel); dir != "." {
				newRel = dir + "/" + name
			}
		}
		newAbs, err := s.targetPath(newRel)
		if err != nil {
			return Change{}, err
		}
		pathChanged, titleChanged := newRel != rel, title != n.Title
		if !pathChanged && !titleChanged {
			out = Moved{OldPath: rel, Path: rel, Title: n.Title, Same: true}
			return Change{}, nil
		}
		if !editableFrontmatter(n) {
			arg := "new_path"
			if op.NewTitle != "" {
				arg = "new_title"
			}
			return Change{}, &ArgError{Arg: arg, Reason: "cannot be applied because the note has no valid frontmatter; add or fix its frontmatter with note_edit first"}
		}
		if pathChanged {
			for _, p := range idx.Paths() {
				if p != rel && strings.EqualFold(p, newRel) {
					return Change{}, &ExistsError{Path: p}
				}
			}
			if !strings.EqualFold(newRel, rel) {
				if _, err := os.Lstat(newAbs); err == nil {
					return Change{}, &ExistsError{Path: newRel}
				}
			}
		}

		data, err := os.ReadFile(oldAbs)
		if err != nil {
			return Change{}, err
		}
		cur := hashOf(data)
		edited, err := parse.EditFrontmatter(data, moveFrontmatter(n, rel, newRel, title, s.stamp())...)
		if err != nil {
			return Change{}, &ArgError{Arg: "new_path", Reason: "cannot be applied because the note's frontmatter cannot be edited; fix it with note_edit first"}
		}
		linkers := idx.Linkers(rel)

		if pathChanged {
			if err := s.renameNote(rel, oldAbs, newAbs, cur); err != nil {
				return Change{}, err
			}
		}
		m, err := s.putFile(newRel, edited, cur)
		if err != nil {
			if pathChanged {
				os.Rename(newAbs, oldAbs) // put the note back where it was
			}
			return Change{}, err
		}
		paths := []string{rel}
		if pathChanged {
			idx.Remove(rel)
			paths = append(paths, newRel)
		}
		out = Moved{OldPath: rel, Path: newRel, Title: title, SHA256: m.SHA256, Bytes: len(edited), Linkers: linkers}
		return Change{Tool: "note_move", Title: title, Paths: paths}, nil
	})
	if err != nil {
		return Moved{}, err
	}
	return out, nil
}

// moveFrontmatter lists the frontmatter changes of a move (SPEC-0007 R20,
// R22): title when it changes, or when it came from the old file name and
// the name changes; the old path without .md and the old title appended to
// aliases when they changed and are not there yet; updated.
func moveFrontmatter(n parse.Note, rel, newRel, title, now string) []parse.FrontmatterSet {
	var sets []parse.FrontmatterSet
	titleChanged := title != n.Title
	if titleChanged || (!n.TitleFromFrontmatter && stem(rel) != stem(newRel)) {
		sets = append(sets, parse.FrontmatterSet{Key: "title", Value: parse.FormatScalar(title)})
	}
	aliases := append([]string(nil), n.Aliases...)
	add := func(a string) {
		for _, x := range aliases {
			if x == a {
				return
			}
		}
		aliases = append(aliases, a)
	}
	if newRel != rel {
		add(strings.TrimSuffix(rel, ".md"))
	}
	if titleChanged {
		add(n.Title)
	}
	if len(aliases) != len(n.Aliases) {
		sets = append(sets, parse.FrontmatterSet{Key: "aliases", Value: parse.FormatList(aliases)})
	}
	return append(sets, parse.FrontmatterSet{Key: "updated", Value: now})
}

func stem(rel string) string { return strings.TrimSuffix(path.Base(rel), ".md") }

// renameNote renames the note at oldAbs to newAbs after checking that it is
// unchanged since the freshness check (SPEC-0007 R3, R22). Missing folders
// of the target are created (R6).
func (s *Service) renameNote(rel, oldAbs, newAbs, prevSHA string) error {
	if err := s.mkdirs(filepath.Dir(newAbs)); err != nil {
		return err
	}
	if s.hooks.beforeRehash != nil {
		s.hooks.beforeRehash(oldAbs)
	}
	if err := checkUnchanged(oldAbs, prevSHA); err != nil {
		return err
	}
	if !strings.EqualFold(oldAbs, newAbs) {
		if _, err := os.Lstat(newAbs); !errors.Is(err, fs.ErrNotExist) {
			return ErrConflict // a file appeared at the target meanwhile
		}
	}
	if err := os.Rename(oldAbs, newAbs); err != nil {
		return err
	}
	syncDir(filepath.Dir(oldAbs))
	return nil
}
