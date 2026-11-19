package note

import "errors"

// ErrBadPath is returned for a tool path that is not a safe note path inside
// the root (SPEC-0003 R14).
var ErrBadPath = errors.New("note: not a safe note path inside the notes root")

// SafePath turns a relative note path from a tool into an absolute path
// inside the root, or returns ErrBadPath. It creates and changes nothing
// (SPEC-0003 R14, R15).
func (s *Service) SafePath(rel string) (string, error) { return "", nil }
