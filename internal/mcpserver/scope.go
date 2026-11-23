package mcpserver

// withScope returns text followed by a line break and the scope line, or
// text unchanged when scope is "" (SPEC-0011 R5–R7).
func withScope(text, scope string) string {
	if scope == "" {
		return text
	}
	return text + "\n" + scopePrefix + scope
}
