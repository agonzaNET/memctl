package note

// Ignored reports whether rel under root is ignored by its .gitignore files,
// reading them the way a walk does.
func Ignored(root, rel string, isDir bool) bool {
	r := loadIgnoreRulesFor(root, rel)
	return r.ignored(rel, isDir)
}

// LockFd returns the file descriptor of the lock file.
func LockFd(l *Lock) uintptr { return l.f.Fd() }
