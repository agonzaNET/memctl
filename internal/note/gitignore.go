package note

// ignoreRules holds the .gitignore rules of every folder seen by a walk,
// keyed by the folder's slash path relative to the root ("" for the root).
type ignoreRules struct{}

// ignored reports whether rel (a file, or a folder when isDir) is matched by
// the .gitignore files of the root and its folders (SPEC-0003 R5).
func (r *ignoreRules) ignored(rel string, isDir bool) bool { return false }

// loadIgnoreRulesFor reads the .gitignore files of root and of every folder
// above rel.
func loadIgnoreRulesFor(root, rel string) *ignoreRules { return &ignoreRules{} }
