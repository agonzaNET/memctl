package note

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// ErrBadPath is returned for a tool path that is not a safe note path inside
// the root (SPEC-0003 R14).
var ErrBadPath = errors.New("note: not a safe note path inside the notes root")

// SafePath turns a relative note path from a tool into an absolute path
// inside the root, or returns ErrBadPath. It creates and changes nothing
// (SPEC-0003 R14, R15).
func (s *Service) SafePath(rel string) (string, error) {
	if rel == "" || strings.HasPrefix(rel, "/") || strings.ContainsAny(rel, "\\\x00") || !strings.HasSuffix(rel, ".md") {
		return "", ErrBadPath
	}
	segs := strings.Split(rel, "/")
	for _, seg := range segs {
		if seg == "" || seg == "." || seg == ".." || strings.HasPrefix(seg, ".") {
			return "", ErrBadPath
		}
	}
	// No existing folder on the way may be a symbolic link.
	dir := s.root
	for _, seg := range segs[:len(segs)-1] {
		dir = filepath.Join(dir, seg)
		fi, err := os.Lstat(dir)
		if errors.Is(err, os.ErrNotExist) {
			break
		}
		if err != nil || fi.Mode()&os.ModeSymlink != 0 || !fi.IsDir() {
			return "", ErrBadPath
		}
	}
	return filepath.Join(s.root, filepath.FromSlash(rel)), nil
}
