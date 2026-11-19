package main

import (
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var update = flag.Bool("update", false, "rewrite golden files")

// checkFixture writes a notes root with one example of every finding.
func checkFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"ok.md":        "---\ntitle: OK\n---\nfine\n",
		"bf.md":        "---\ntitle: [x\n---\nbody\n",
		"bfield.md":    "---\ncreated: soon\n---\n",
		"bl.md":        "x [[]]\n",
		"misplaced.md": "- [risk] in the body\n- implements [[ok]]\n",
		"forms.md":     "Title\n=====\n\n<div>\n</div>\n\n    code\n",
		"links.md":     "[[nowhere]] [[dup]] [[old/place]]\n",
		"x/dup.md":     "x\n",
		"y/dup.md":     "y\n",
		"target.md":    "---\naliases: [old/place]\n---\n",
		"utf.md":       "bad \xff byte\n",
		"big.md":       strings.Repeat("x", 4<<20+1),
	}
	for p, c := range files {
		full := filepath.Join(root, filepath.FromSlash(p))
		os.MkdirAll(filepath.Dir(full), 0o755)
		if err := os.WriteFile(full, []byte(c), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestCheck_SPEC0005_R2_R3_R4_R5_golden(t *testing.T) {
	root := checkFixture(t)
	r := runBin(t, binVersioned, nil, "check", "--root", root)
	golden := filepath.Join("testdata", "check.golden")
	if *update {
		os.WriteFile(golden, []byte(r.stdout), 0o644)
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if r.stdout != string(want) {
		t.Fatalf("report differs from %s:\n%s", golden, r.stdout)
	}
	if r.code != 1 {
		t.Fatalf("exit %d, want 1 (errors present)", r.code)
	}
	for _, code := range []string{
		"bad_frontmatter", "bad_field", "bad_link", "observation_outside_section", "relation_outside_section",
		"setext_heading", "html_block", "indented_code", "invalid_utf8", "dangling_link", "ambiguous_link",
		"stale_link", "skipped_file", "no_git_repo",
	} {
		if !strings.Contains(r.stdout, ": "+code+": ") {
			t.Errorf("report has no %s", code)
		}
	}
}

func TestCheck_SPEC0005_R6_exitStatuses(t *testing.T) {
	clean := t.TempDir()
	os.WriteFile(filepath.Join(clean, "a.md"), []byte("---\ntitle: A\n---\n[[b]]\n"), 0o644)
	os.WriteFile(filepath.Join(clean, "b.md"), []byte("b\n"), 0o644)
	gitInit(t, clean)
	r := runBin(t, binVersioned, nil, "check", "--root", clean)
	if r.code != 0 || r.stdout != "2 notes checked, 0 errors, 0 warnings.\n" {
		t.Fatalf("clean: %+v", r)
	}

	warn := t.TempDir()
	os.WriteFile(filepath.Join(warn, "a.md"), []byte("[[missing]]\n"), 0o644)
	r = runBin(t, binVersioned, nil, "check", "--root", warn)
	if r.code != 0 || !strings.HasSuffix(r.stdout, "1 note checked, 0 errors, 2 warnings.\n") {
		t.Fatalf("warnings only: %+v", r)
	}

	r = runBin(t, binVersioned, nil, "check", "--root", clean, "--bogus")
	if r.code != 2 {
		t.Fatalf("bad flag: %+v", r)
	}
	r = runBin(t, binVersioned, nil, "check", "--root", filepath.Join(clean, "missing"))
	if r.code != 1 || r.stdout != "" || !strings.Contains(r.stderr, `"level":"ERROR"`) {
		t.Fatalf("missing root: %+v", r)
	}
}

func gitInit(t *testing.T, dir string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	if out, err := exec.Command("git", "init", "-q", dir).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, out)
	}
}

func TestCheck_SPEC0005_R1_R8_readOnlyAndCleanStdout(t *testing.T) {
	root := checkFixture(t)
	gitInit(t, root)
	before := stateOf(t, root)
	statusBefore, _ := exec.Command("git", "-C", root, "status", "--porcelain").Output()

	// A memctl mcp serving the same root does not stop the check.
	mcp := exec.Command(binVersioned, "mcp", "--root", root)
	mcpIn, _ := mcp.StdinPipe()
	if err := mcp.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { mcpIn.Close(); mcp.Wait() }()
	time.Sleep(200 * time.Millisecond)

	r := runBin(t, binVersioned, nil, "check", "--root", root)
	if r.code != 1 {
		t.Fatalf("check: %+v", r)
	}
	for _, l := range strings.Split(strings.TrimSuffix(r.stdout, "\n"), "\n") {
		if !strings.Contains(l, ": ") && !strings.Contains(l, " notes checked, ") {
			t.Fatalf("stdout line is neither a finding nor the summary: %q", l)
		}
	}
	for _, l := range jsonLines(t, r.stderr) {
		if l["level"] == "INFO" || l["level"] == "DEBUG" {
			t.Fatalf("check logged below warn: %v", l)
		}
	}
	if after := stateOf(t, root); after != before {
		t.Fatalf("root changed by check")
	}
	statusAfter, _ := exec.Command("git", "-C", root, "status", "--porcelain").Output()
	if string(statusAfter) != string(statusBefore) {
		t.Fatalf("git status changed:\n%s\n%s", statusBefore, statusAfter)
	}
}

// stateOf lists every path under root with its size and modification time.
func stateOf(t *testing.T, root string) string {
	t.Helper()
	var b strings.Builder
	filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil || strings.Contains(p, string(filepath.Separator)+".git") {
			return nil
		}
		b.WriteString(p + " " + info.ModTime().String() + "\n")
		return nil
	})
	return b.String()
}
