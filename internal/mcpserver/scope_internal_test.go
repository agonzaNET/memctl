package mcpserver

import (
	"os"
	"strings"
	"testing"
)

// TestScope_SPEC0011_R9_prefixIsAgentFacingText checks that the fixed part of
// the scope line sits between the markers that scripts/check-docs.sh scans.
func TestScope_SPEC0011_R9_prefixIsAgentFacingText(t *testing.T) {
	src, err := os.ReadFile("tools.go")
	if err != nil {
		t.Fatal(err)
	}
	s := string(src)
	begin := strings.Index(s, "// agent-facing text: begin\n")
	end := strings.Index(s, "// agent-facing text: end\n")
	at := strings.Index(s, "const scopePrefix = `Scope of this memory, set by the person who runs it: `")
	if begin < 0 || end < 0 || at < begin || at > end {
		t.Fatalf("scopePrefix is not between the agent-facing markers (begin %d, at %d, end %d)", begin, at, end)
	}
}
