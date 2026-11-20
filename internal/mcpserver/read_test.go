package mcpserver_test

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const exampleNote = `---
title: PAN-OS upgrade procedure
type: runbook
tags: [palo-alto, firewall]
aliases: [PAN-OS upgrade]
created: 2026-11-19T10:12:00+01:00
updated: 2026-11-19T10:12:00+01:00
---

# PAN-OS upgrade procedure

Free Markdown body. A mention of [[PA-850]] here is indexed as links_to.

## Observations
- [prereq] Download content updates before the base image
- [risk] HA pair must be upgraded passive first

## Relations
- implements [[ODR-017]]
- applies_to [[PA-850]]
`

func sum(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func TestRead_SPEC0004_R15_R17_example(t *testing.T) {
	const rel = "runbooks/PAN-OS upgrade procedure.md"
	e := start(t, map[string]string{rel: exampleNote})
	got := text(t, e.call(t, "note_read", map[string]any{"ref": "runbooks/PAN-OS upgrade procedure"}))
	want := `Read "PAN-OS upgrade procedure" at runbooks/PAN-OS upgrade procedure.md.

- path: runbooks/PAN-OS upgrade procedure.md
- title: PAN-OS upgrade procedure
- type: runbook
- tags: palo-alto, firewall
- aliases: PAN-OS upgrade
- created: 2026-11-19T10:12:00+01:00
- updated: 2026-11-19T10:12:00+01:00
- sha256: ` + sum(exampleNote) + "\n\n````markdown\n" + exampleNote + "````\n\nLinks:"
	if !strings.HasPrefix(got, want) {
		t.Fatalf("got:\n%s\n\nwant prefix:\n%s", got, want)
	}
}

func TestRead_SPEC0004_R15_fenceLongerThanContent(t *testing.T) {
	content := "---\ntitle: F\n---\n````go\ncode\n````\n"
	e := start(t, map[string]string{"f.md": content})
	got := text(t, e.call(t, "note_read", map[string]any{"ref": "f.md"}))
	if !strings.Contains(got, "\n`````markdown\n"+content+"`````\n") {
		t.Fatalf("fence: %s", got)
	}
	e2 := start(t, map[string]string{"g.md": "plain\n"})
	got = text(t, e2.call(t, "note_read", map[string]any{"ref": "g"}))
	if !strings.Contains(got, "\n````markdown\nplain\n````\n") {
		t.Fatalf("minimum fence: %s", got)
	}
	if !strings.Contains(got, "- title: g\n") || strings.Contains(got, "- type:") {
		t.Fatalf("empty fields must be left out: %s", got)
	}
}

func TestRead_SPEC0004_R14_refs(t *testing.T) {
	e := start(t, map[string]string{
		"hosts/intpxy.md":    "---\ntitle: intpxy host\naliases: [proxy box]\n---\n",
		"services/intpxy.md": "---\ntitle: intpxy service\n---\n",
	})
	for _, ref := range []string{"hosts/intpxy", "hosts/intpxy.md", "intpxy host", "Proxy Box"} {
		r := e.call(t, "note_read", map[string]any{"ref": ref})
		if r.IsError || !strings.HasPrefix(text(t, r), `Read "intpxy host" at hosts/intpxy.md.`) {
			t.Fatalf("ref %q: %s", ref, text(t, r))
		}
	}
	r := e.call(t, "note_read", map[string]any{"ref": "intpxy"})
	if !r.IsError || !strings.HasPrefix(text(t, r), "ambiguous: ") ||
		!strings.Contains(text(t, r), "hosts/intpxy.md") || !strings.Contains(text(t, r), "services/intpxy.md") {
		t.Fatalf("ambiguous: %s", text(t, r))
	}
	r = e.call(t, "note_read", map[string]any{"ref": "nowhere"})
	if !r.IsError || !strings.HasPrefix(text(t, r), "not_found: ") || !strings.Contains(text(t, r), "search") {
		t.Fatalf("not found: %s", text(t, r))
	}
}

func TestRead_SPEC0004_R7_freshContent(t *testing.T) {
	e := start(t, map[string]string{"a.md": "first\n"})
	got := text(t, e.call(t, "note_read", map[string]any{"ref": "a"}))
	if !strings.Contains(got, sum("first\n")) {
		t.Fatalf("first read: %s", got)
	}
	p := filepath.Join(e.root, "a.md")
	os.WriteFile(p, []byte("second, longer\n"), 0o644)
	later := time.Now().Add(time.Minute)
	os.Chtimes(p, later, later)
	got = text(t, e.call(t, "note_read", map[string]any{"ref": "a"}))
	if !strings.Contains(got, "second, longer") || !strings.Contains(got, sum("second, longer\n")) {
		t.Fatalf("edit not seen: %s", got)
	}
}

func TestRead_SPEC0004_R18_counts(t *testing.T) {
	e := start(t, map[string]string{"a.md": "12345\n"})
	e.call(t, "note_read", map[string]any{"ref": "a"})
	l := e.logLines(t, "tool call")
	if len(l) != 1 || l[0]["bytes"] != 6.0 || l[0]["links"] != 0.0 {
		t.Fatalf("counts %v", l)
	}
}

func TestRead_SPEC0004_R16_section(t *testing.T) {
	const rel = "runbooks/PAN-OS upgrade procedure.md"
	e := start(t, map[string]string{rel: exampleNote})
	got := text(t, e.call(t, "note_read", map[string]any{"ref": rel, "section": " observations "}))
	wantSection := "## Observations\n- [prereq] Download content updates before the base image\n- [risk] HA pair must be upgraded passive first\n\n"
	if !strings.HasPrefix(got, `Read section "Observations" of "PAN-OS upgrade procedure" at runbooks/PAN-OS upgrade procedure.md.`) {
		t.Fatalf("first line: %s", got)
	}
	if !strings.Contains(got, "````markdown\n"+wantSection+"````") {
		t.Fatalf("section content: %s", got)
	}
	if !strings.Contains(got, "- sha256: "+sum(exampleNote)) {
		t.Fatalf("sha256 must be the whole file's: %s", got)
	}

	// A level-1 section runs over its sub-headings to the next level-1 heading.
	got = text(t, e.call(t, "note_read", map[string]any{"ref": rel, "section": "PAN-OS upgrade procedure"}))
	if !strings.Contains(got, "## Relations\n- implements [[ODR-017]]\n- applies_to [[PA-850]]\n````") {
		t.Fatalf("level-1 section: %s", got)
	}

	r := e.call(t, "note_read", map[string]any{"ref": rel, "section": "Missing"})
	if !r.IsError || !strings.HasPrefix(text(t, r), "section_not_found: ") || !strings.Contains(text(t, r), "Observations") ||
		!strings.Contains(text(t, r), "pass one of them") {
		t.Fatalf("missing section: %s", text(t, r))
	}
}

func TestRead_SPEC0004_R16_sectionLinks(t *testing.T) {
	const rel = "runbooks/PAN-OS upgrade procedure.md"
	e := start(t, map[string]string{rel: exampleNote})
	// The Relations section holds two relations; the body link above it is
	// not listed.
	got := text(t, e.call(t, "note_read", map[string]any{"ref": rel, "section": "Relations"}))
	want := "Links in this section:\n- implements → ODR-017: dangling\n- applies_to → PA-850: dangling"
	if !strings.HasSuffix(got, want) || strings.Contains(got, "links_to") {
		t.Fatalf("got:\n%s", got)
	}
	got = text(t, e.call(t, "note_read", map[string]any{"ref": rel, "section": "Observations"}))
	if !strings.HasSuffix(got, "Links in this section: none.") {
		t.Fatalf("got:\n%s", got)
	}
	// Without section, the whole note's list is unchanged.
	got = text(t, e.call(t, "note_read", map[string]any{"ref": rel}))
	if !strings.Contains(got, "\nLinks:\n- links_to → PA-850: dangling") {
		t.Fatalf("got:\n%s", got)
	}
}

func TestRead_SPEC0004_R15_linksList(t *testing.T) {
	const rel = "runbooks/PAN-OS upgrade procedure.md"
	e := start(t, map[string]string{
		rel:                    exampleNote,
		"decisions/ODR-017.md": "---\ntitle: ODR-017 upgrade cadence\n---\n",
		"hosts/PA-850.md":      "---\ntitle: PA-850\naliases: [old/fw]\n---\n",
		"services/PA-850.md":   "---\ntitle: PA-850\n---\n",
	})
	got := text(t, e.call(t, "note_read", map[string]any{"ref": rel}))
	want := "Links:\n" +
		"- links_to → PA-850: ambiguous (hosts/PA-850.md, services/PA-850.md)\n" +
		"- implements → decisions/ODR-017.md\n" +
		"- applies_to → PA-850: ambiguous (hosts/PA-850.md, services/PA-850.md)"
	if !strings.HasSuffix(got, want) {
		t.Fatalf("links list:\n%s", got)
	}
	l := e.logLines(t, "tool call")
	if l[len(l)-1]["links"] != 3.0 {
		t.Fatalf("links count %v", l)
	}

	e2 := start(t, map[string]string{
		"a.md":       "Body [[old/fw]] and [[nowhere]]\n",
		"hosts/x.md": "---\ntitle: X\naliases: [old/fw]\n---\n",
	})
	got = text(t, e2.call(t, "note_read", map[string]any{"ref": "a"}))
	want = "Links:\n- links_to → hosts/x.md (stale: linked as old/fw, an old path)\n- links_to → nowhere: dangling"
	if !strings.HasSuffix(got, want) {
		t.Fatalf("stale and dangling:\n%s", got)
	}
}

func TestRead_SPEC0004_R28_suggestions(t *testing.T) {
	files := map[string]string{
		"misc/other.md": "---\ntitle: Other\n---\nnothing\n",
	}
	for i := 0; i < 5; i++ {
		files[fmt.Sprintf("runbooks/r%d.md", i)] = "---\ntitle: R\n---\nswitch upgrade\n"
	}
	e := start(t, files)
	r := e.call(t, "note_read", map[string]any{"ref": "Upgrade the secret switch"})
	want := `not_found: No note matches "Upgrade the secret switch"; closest by search: runbooks/r0.md, runbooks/r1.md, runbooks/r2.md; pass one of these paths, or use search.`
	if !r.IsError || text(t, r) != want {
		t.Fatalf("got %q", text(t, r))
	}
	r = e.call(t, "context", map[string]any{"ref": "zzz qqq"})
	want = `not_found: No note matches "zzz qqq"; use search to find the note, then pass its path.`
	if !r.IsError || text(t, r) != want {
		t.Fatalf("got %q", text(t, r))
	}
	if strings.Contains(e.log.String(), "secret") {
		t.Fatalf("ref in log: %s", e.log.String())
	}
}

func TestRead_SPEC0004_R28_nearNames(t *testing.T) {
	e := start(t, map[string]string{
		"hosts/edge router.md": "---\ntitle: Edge router\n---\nModel cisco1 in rack 4.\n",
		"misc/other.md":        "---\ntitle: Other\n---\nnothing\n",
	})
	r := e.call(t, "note_read", map[string]any{"ref": "cisco"})
	want := `not_found: No note matches "cisco"; closest by search: hosts/edge router.md; pass one of these paths, or use search.`
	if !r.IsError || text(t, r) != want {
		t.Fatalf("got %q", text(t, r))
	}
	// search finds no notes; since R32 it names the near name as a word,
	// never the note.
	r = e.call(t, "search", map[string]any{"query": "cisco"})
	if got := text(t, r); r.IsError || !strings.HasPrefix(got, `Found no notes for "cisco".`) || strings.Contains(got, "edge router") {
		t.Fatalf("search must not use near names: %q", got)
	}
	if strings.Contains(e.log.String(), "cisco") {
		t.Fatalf("ref in log: %s", e.log.String())
	}
}
