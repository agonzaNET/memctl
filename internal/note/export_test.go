package note

// Ignored reports whether rel under root is ignored by its .gitignore files,
// reading them the way a walk does.
func Ignored(root, rel string, isDir bool) bool {
	r := loadIgnoreRulesFor(root, rel)
	return r.ignored(rel, isDir)
}
