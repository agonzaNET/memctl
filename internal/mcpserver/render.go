package mcpserver

import (
	"fmt"
	"strings"

	"github.com/agonzaNET/memctl/internal/index"
)

// fence returns a backtick fence one longer than the longest run of
// backticks in content, and at least four (SPEC-0004 R15).
func fence(content string) string {
	longest, run := 0, 0
	for _, r := range content {
		if r == '`' {
			run++
			if run > longest {
				longest = run
			}
		} else {
			run = 0
		}
	}
	n := longest + 1
	if n < 4 {
		n = 4
	}
	return strings.Repeat("`", n)
}

// fenced wraps content in a markdown fence that it cannot close.
func fenced(content string) string {
	f := fence(content)
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return f + "markdown\n" + content + f
}

// resolveRef resolves a tool ref, or returns not_found or ambiguous
// (SPEC-0004 R14).
func resolveRef(idx *index.Index, ref string) (string, error) {
	r := idx.Resolve(ref)
	switch r.Status {
	case index.Resolved:
		return r.Path, nil
	case index.Ambiguous:
		return "", errorf(CodeAmbiguous, "%q matches several notes: %s; call again with one of these paths as ref.", ref, strings.Join(r.Candidates, ", "))
	default:
		return "", errorf(CodeNotFound, "No note matches %q; use search to find the note, then pass its path.", ref)
	}
}

// describe renders one resolved link target for a links list.
func describe(target string, r index.Resolution) string {
	switch r.Status {
	case index.Resolved:
		if r.Stale {
			return fmt.Sprintf("%s (stale: linked as %s, an old path)", r.Path, target)
		}
		return r.Path
	case index.Ambiguous:
		return fmt.Sprintf("%s: ambiguous (%s)", target, strings.Join(r.Candidates, ", "))
	default:
		return target + ": dangling"
	}
}
