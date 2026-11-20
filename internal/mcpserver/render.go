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

// resolveRef resolves a tool ref to a path, or returns not_found or
// ambiguous (SPEC-0004 R14).
func resolveRef(idx *index.Index, ref string) (string, error) {
	r, err := resolveRefFull(idx, ref)
	return r.Path, err
}

// resolveRefFull is resolveRef with the whole resolution, for the tools
// that say how a ref resolved (SPEC-0004 R27).
func resolveRefFull(idx *index.Index, ref string) (index.Resolution, error) {
	r := idx.Resolve(ref)
	switch r.Status {
	case index.Resolved:
		return r, nil
	case index.Ambiguous:
		return r, errorf(CodeAmbiguous, "%q matches several notes: %s; call again with one of these paths as ref.", ref, strings.Join(r.Candidates, ", "))
	default:
		return r, notFound(idx, ref)
	}
}

// notFound is the not_found error of an unknown ref, with up to three
// paths that a search for the ref's words finds (SPEC-0004 R28).
func notFound(idx *index.Index, ref string) error {
	found, err := idx.Search(index.Query{Text: ref, Limit: 3})
	if err != nil || len(found.Hits) == 0 {
		return errorf(CodeNotFound, "No note matches %q; use search to find the note, then pass its path.", ref)
	}
	paths := make([]string, len(found.Hits))
	for i, h := range found.Hits {
		paths[i] = h.Path
	}
	return errorf(CodeNotFound, "No note matches %q; closest by search: %s; pass one of these paths, or use search.", ref, strings.Join(paths, ", "))
}

// resolvedBy returns how a ref resolved when not by path, and the other
// notes with that name, or empty strings (SPEC-0004 R27).
func resolvedBy(ref string, r index.Resolution) (step, others string) {
	if r.Step != index.StepPath {
		step = fmt.Sprintf("%s %q", r.Step, ref)
	}
	return step, strings.Join(r.Also, ", ")
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
