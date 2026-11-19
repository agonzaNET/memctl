package note

import (
	"io/fs"
	"path/filepath"
	"strings"
	"time"
)

// candidate is a file the walk found that may be a note.
type candidate struct {
	rel     string // slash-separated, relative to the root
	abs     string
	size    int64
	modTime time.Time
}

// walkResult is what one walk of the root found.
type walkResult struct {
	notes      []candidate
	unreadable []string // relative paths of unreadable files or folders
}

// walk lists the notes under root (SPEC-0003 R3–R5): regular .md files whose
// name does not start with a dot, outside dot-folders and outside what the
// .gitignore files match, without following symbolic links. Unreadable entries are reported, not fatal.
func walk(root string) walkResult {
	var res walkResult
	rules := newIgnoreRules()
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		rel := relSlash(root, p)
		if err != nil {
			if p != root {
				res.unreadable = append(res.unreadable, rel)
			}
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		name := d.Name()
		if d.IsDir() {
			if p == root {
				rules.load(root, "")
				return nil
			}
			if strings.HasPrefix(name, ".") || rules.ignored(rel, true) {
				return fs.SkipDir
			}
			rules.load(root, rel)
			return nil
		}
		if !d.Type().IsRegular() || strings.HasPrefix(name, ".") || !strings.HasSuffix(name, ".md") {
			return nil
		}
		if rules.ignored(rel, false) {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			res.unreadable = append(res.unreadable, rel)
			return nil
		}
		res.notes = append(res.notes, candidate{rel: rel, abs: p, size: info.Size(), modTime: info.ModTime()})
		return nil
	})
	return res
}

func relSlash(root, p string) string {
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return filepath.ToSlash(p)
	}
	return filepath.ToSlash(rel)
}
