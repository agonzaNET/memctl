package note

import "github.com/agonzaNET/memctl/internal/index"

// Ignored reports whether rel under root is ignored by its .gitignore files,
// reading them the way a walk does.
func Ignored(root, rel string, isDir bool) bool {
	r := loadIgnoreRulesFor(root, rel)
	return r.ignored(rel, isDir)
}

// LockFd returns the file descriptor of the lock file.
func LockFd(l *Lock) uintptr { return l.f.Fd() }

// Mutate runs fn as a tool write.
func Mutate(s *Service, fn func(idx *index.Index) (Change, error)) error { return s.mutate(fn) }

// PutFile writes a note atomically; call it inside Mutate.
func PutFile(s *Service, rel string, data []byte, prevSHA string) (index.Meta, error) {
	return s.putFile(rel, data, prevSHA)
}

// RemoveFile removes a note; call it inside Mutate.
func RemoveFile(s *Service, rel, prevSHA string) error { return s.removeFile(rel, prevSHA) }

// TargetPath checks a path a tool wants to write.
func TargetPath(s *Service, rel string) (string, error) { return s.targetPath(rel) }

// Stamp formats the service clock's now.
func Stamp(s *Service) string { return s.stamp() }

// SetHooks installs fault-injection hooks.
func SetHooks(s *Service, beforeRehash func(abs string), beforeRename func(abs string) error) {
	s.hooks = hooks{beforeRehash: beforeRehash, beforeRename: beforeRename}
}
