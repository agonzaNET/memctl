package note

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/agonzaNET/memctl/internal/index"
	"github.com/agonzaNET/memctl/internal/parse"
)

// ErrConflict is returned when a note changed after the freshness check of
// the call, so a write would overwrite a state the agent has not seen
// (SPEC-0007 R3, R15).
var ErrConflict = errors.New("note: the file changed since it was read")

// PathError is a tool path that cannot be a note, with the reason in words
// for the agent (SPEC-0007 R28). It wraps ErrBadPath.
type PathError struct{ Path, Reason string }

func (e *PathError) Error() string { return "path " + e.Path + ": " + e.Reason }
func (e *PathError) Unwrap() error { return ErrBadPath }

// stampLayout is the layout of created and updated (SPEC-0007 R7).
const stampLayout = "2006-01-02T15:04:05-07:00"

// stamp is now from the service clock, to the second, with its offset.
func (s *Service) stamp() string { return s.now().Format(stampLayout) }

// mutate runs one tool write: the freshness check and fn under the service
// mutex (SPEC-0007 R1). When fn succeeds with changed paths, the change goes
// to the committer; a failed write hands nothing (R5).
func (s *Service) mutate(fn func(idx *index.Index) (Change, error)) error {
	var ch Change
	err := s.Call(func(idx *index.Index, _ Stats) error {
		var err error
		ch, err = fn(idx)
		return err
	})
	if err != nil {
		return err
	}
	if len(ch.Paths) > 0 {
		s.committer.Enqueue(ch)
	}
	return nil
}

// targetPath checks a path that a tool wants to write and returns it as an
// absolute path. It refuses what SafePath refuses, and what the walk would
// not treat as a note, so the note stays visible (SPEC-0003 R14, SPEC-0007
// R28).
func (s *Service) targetPath(rel string) (string, error) {
	for _, seg := range strings.Split(rel, "/") {
		if strings.HasPrefix(seg, ".") && seg != "." && seg != ".." {
			return "", &PathError{rel, `a folder or file name that starts with "." is never read as a note`}
		}
	}
	abs, err := s.SafePath(rel)
	if err != nil {
		if !strings.HasSuffix(rel, ".md") {
			return "", &PathError{rel, "a note path must end in .md"}
		}
		return "", &PathError{rel, "a note path must be relative, inside the notes root, without empty, \".\" or \"..\" parts, and must not pass through a symbolic link"}
	}
	rules := loadIgnoreRulesFor(s.root, rel)
	if dir := path.Dir(rel); dir != "." {
		parts := strings.Split(dir, "/")
		for i := range parts {
			if rules.ignored(strings.Join(parts[:i+1], "/"), true) {
				return "", &PathError{rel, "a .gitignore file matches the folder " + strings.Join(parts[:i+1], "/") + ", so the note would never be read"}
			}
		}
	}
	if rules.ignored(rel, false) {
		return "", &PathError{rel, "a .gitignore file matches this path, so the note would never be read"}
	}
	return abs, nil
}

// putFile writes data to the note rel atomically and updates its index entry
// (SPEC-0007 R2, R4, R6). prevSHA is the hash the file had at the freshness
// check, or "" for a note that must not exist yet; any other state just
// before the rename gives ErrConflict (R3). It must run inside mutate.
func (s *Service) putFile(rel string, data []byte, prevSHA string) (index.Meta, error) {
	abs, err := s.targetPath(rel)
	if err != nil {
		return index.Meta{}, err
	}
	dir := filepath.Dir(abs)
	if err := s.mkdirs(dir); err != nil {
		return index.Meta{}, err
	}
	perm := fs.FileMode(0o644)
	if fi, err := os.Stat(abs); err == nil {
		perm = fi.Mode().Perm()
	}

	tmp, err := os.CreateTemp(dir, ".memctl-*.tmp")
	if err != nil {
		return index.Meta{}, err
	}
	tmpName := tmp.Name()
	done := false
	defer func() {
		if !done {
			os.Remove(tmpName)
		}
	}()
	_, werr := tmp.Write(data)
	if werr == nil {
		werr = tmp.Sync()
	}
	if werr == nil {
		werr = tmp.Chmod(perm)
	}
	if cerr := tmp.Close(); werr == nil {
		werr = cerr
	}
	if werr != nil {
		return index.Meta{}, werr
	}

	if s.hooks.beforeRehash != nil {
		s.hooks.beforeRehash(abs)
	}
	if err := checkUnchanged(abs, prevSHA); err != nil {
		return index.Meta{}, err
	}
	if s.hooks.beforeRename != nil {
		if err := s.hooks.beforeRename(abs); err != nil {
			return index.Meta{}, err
		}
	}
	readAt := time.Now() // the content is known from here on (SPEC-0003 R17)
	if err := os.Rename(tmpName, abs); err != nil {
		return index.Meta{}, err
	}
	done = true
	if err := syncDir(dir); err != nil {
		s.log.Warn("cannot sync the folder after a write", slog.String("err", err.Error()))
	}

	fi, err := os.Stat(abs)
	if err != nil {
		return index.Meta{}, err
	}
	h := sha256.Sum256(data)
	m := index.Meta{ModTime: fi.ModTime(), Size: int64(len(data)), SHA256: hex.EncodeToString(h[:]), ReadAt: readAt}
	s.idx.Put(rel, parse.Parse(rel, data), m)
	return m, nil
}

// removeFile deletes the note rel and its index entry (SPEC-0007 R4, R24),
// after checking that the file still has hash prevSHA (R3). Empty folders
// stay (R6). It must run inside mutate.
func (s *Service) removeFile(rel, prevSHA string) error {
	abs, err := s.SafePath(rel)
	if err != nil {
		return &PathError{rel, "not a note path inside the notes root"}
	}
	if s.hooks.beforeRehash != nil {
		s.hooks.beforeRehash(abs)
	}
	if err := checkUnchanged(abs, prevSHA); err != nil {
		return err
	}
	if s.hooks.beforeRename != nil {
		if err := s.hooks.beforeRename(abs); err != nil {
			return err
		}
	}
	if err := os.Remove(abs); err != nil {
		return err
	}
	if err := syncDir(filepath.Dir(abs)); err != nil {
		s.log.Warn("cannot sync the folder after a delete", slog.String("err", err.Error()))
	}
	s.idx.Remove(rel)
	return nil
}

// checkUnchanged reports ErrConflict unless abs has hash prevSHA, or does not
// exist when prevSHA is "" (SPEC-0007 R3).
func checkUnchanged(abs, prevSHA string) error {
	data, err := os.ReadFile(abs)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		if prevSHA == "" {
			return nil
		}
		return ErrConflict
	case err != nil:
		return err
	case prevSHA == "":
		return ErrConflict
	}
	h := sha256.Sum256(data)
	if hex.EncodeToString(h[:]) != prevSHA {
		return ErrConflict
	}
	return nil
}

// mkdirs creates the missing folders of dir with mode 0755 (SPEC-0007 R6).
func (s *Service) mkdirs(dir string) error {
	rel, err := filepath.Rel(s.root, dir)
	if err != nil || rel == "." {
		return err
	}
	cur := s.root
	for _, seg := range strings.Split(rel, string(filepath.Separator)) {
		cur = filepath.Join(cur, seg)
		err := os.Mkdir(cur, 0o755)
		switch {
		case err == nil:
			if err := os.Chmod(cur, 0o755); err != nil { // the umask may have narrowed it
				return err
			}
		case errors.Is(err, fs.ErrExist):
		default:
			return err
		}
	}
	return nil
}

// syncDir flushes a folder's entries to disk after a rename or remove.
func syncDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	if err := d.Sync(); err != nil && !errors.Is(err, errors.ErrUnsupported) {
		return fmt.Errorf("sync %s: %w", dir, err)
	}
	return nil
}
