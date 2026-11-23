package note_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/agonzaNET/memctl/internal/note"
	"github.com/agonzaNET/memctl/internal/parse"
)

func TestOpen_SPEC0003_R1_root(t *testing.T) {
	root := tree(t, map[string]string{"a.md": "x"})
	var buf bytes.Buffer

	// Relative root.
	wd, _ := os.Getwd()
	rel, err := filepath.Rel(wd, root)
	if err != nil {
		t.Fatal(err)
	}
	s, _, err := note.Open(rel, logger(&buf))
	if err != nil {
		t.Fatalf("relative root: %v", err)
	}
	want, _ := filepath.EvalSymlinks(root)
	if s.Root() != want || !filepath.IsAbs(s.Root()) {
		t.Fatalf("root %q, want %q", s.Root(), want)
	}

	// Root reached through a symlink resolves to the target.
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(root, link); err != nil {
		t.Fatal(err)
	}
	s, _, err = note.Open(link, logger(&buf))
	if err != nil || s.Root() != want {
		t.Fatalf("symlinked root: %v %q", err, s.Root())
	}

	// Failures name the path.
	missing := filepath.Join(root, "missing")
	file := filepath.Join(root, "a.md")
	for _, p := range []string{missing, file} {
		if _, _, err := note.Open(p, logger(&buf)); err == nil || !strings.Contains(err.Error(), p) {
			t.Fatalf("Open(%q) err = %v, want an error naming the path", p, err)
		}
	}
}

func TestOpen_SPEC0003_R1_unreadableRoot(t *testing.T) {
	skipIfRoot(t)
	root := t.TempDir()
	if err := os.Chmod(root, 0); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(root, 0o755)
	var buf bytes.Buffer
	if _, _, err := note.Open(root, logger(&buf)); err == nil || !strings.Contains(err.Error(), root) {
		t.Fatalf("err = %v", err)
	}
}

func TestOpen_SPEC0003_R2_R3_R4_whichFilesAreNotes(t *testing.T) {
	root := tree(t, map[string]string{
		"a.md": "a",
		// An upper-case extension is not a note. The name differs from a.md
		// in more than letter case: on a case-insensitive file system (the
		// macOS default) A.MD and a.md are one file, named by whichever the
		// map wrote first.
		"B.MD":            "upper",
		".hidden.md":      "hidden",
		"x.txt":           "text",
		".git/x.md":       "git",
		".obsidian/y.md":  "obsidian",
		".trash/z.md":     "trash",
		".memory/m.md":    "memory",
		"sub/dir/deep.md": "deep",
		"sub/.d/e.md":     "dot folder",
		"target/t.md":     "target",
	})
	if err := os.Symlink(filepath.Join(root, "target", "t.md"), filepath.Join(root, "linked.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "target"), filepath.Join(root, "linkdir")); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	s, st, err := note.Open(root, logger(&buf))
	if err != nil {
		t.Fatal(err)
	}
	got := s.Index().Paths()
	want := []string{"a.md", "sub/dir/deep.md", "target/t.md"}
	if !reflect.DeepEqual(got, want) || st.Added != 3 {
		t.Fatalf("paths %v (added %d), want %v", got, st.Added, want)
	}
}

func TestOpen_SPEC0003_R6_R7_skipped(t *testing.T) {
	big := strings.Repeat("x", note.MaxNoteSize+1)
	files := map[string]string{"ok.md": "ok", "big.md": big, "big2.md": big}
	root := tree(t, files)
	unreadable := os.Geteuid() != 0
	if unreadable {
		if err := os.WriteFile(filepath.Join(root, "locked.md"), []byte("x"), 0); err != nil {
			t.Fatal(err)
		}
	}
	var buf bytes.Buffer
	s, st, err := note.Open(root, logger(&buf))
	if err != nil {
		t.Fatal(err)
	}
	wantSkipped := 2
	if unreadable {
		wantSkipped = 3
	}
	if st.Skipped != wantSkipped || !reflect.DeepEqual(s.Index().Paths(), []string{"ok.md"}) {
		t.Fatalf("skipped %d paths %v", st.Skipped, s.Index().Paths())
	}
	warns := find(records(t, &buf), "notes skipped")
	reasons := map[string]float64{}
	for _, w := range warns {
		if w["level"] != "WARN" {
			t.Fatalf("level %v", w["level"])
		}
		reasons[w["reason"].(string)] = w["count"].(float64)
	}
	if reasons["too_large"] != 2 || (unreadable && reasons["unreadable"] != 1) {
		t.Fatalf("warn lines %v", warns)
	}
	if strings.Contains(buf.String(), "big.md") && !strings.Contains(buf.String(), `"level":"DEBUG"`) {
		t.Fatalf("a skipped path is logged above DEBUG: %s", buf.String())
	}
}

func TestOpen_SPEC0003_R8_parallelEqualsSequential(t *testing.T) {
	files := map[string]string{}
	for i := 0; i < 500; i++ {
		files[fmt.Sprintf("d%d/n%03d.md", i%7, i)] = fmt.Sprintf("---\ntitle: Note %d\ntags: [t%d]\n---\nBody [[Note %d]]\n", i, i%5, (i+1)%500)
	}
	root := tree(t, files)
	var buf bytes.Buffer
	s, st, err := note.Open(root, logger(&buf))
	if err != nil {
		t.Fatal(err)
	}
	if st.Added != 500 || s.Index().Len() != 500 {
		t.Fatalf("added %d len %d", st.Added, s.Index().Len())
	}
	for rel, content := range files {
		n, m, ok := s.Index().Get(rel)
		if !ok {
			t.Fatalf("missing %s", rel)
		}
		if !reflect.DeepEqual(n, parse.Parse(rel, []byte(content))) {
			t.Fatalf("note %s differs from a sequential parse", rel)
		}
		sum := sha256.Sum256([]byte(content))
		fi, _ := os.Stat(filepath.Join(root, filepath.FromSlash(rel)))
		if m.SHA256 != hex.EncodeToString(sum[:]) || m.Size != int64(len(content)) || !m.ModTime.Equal(fi.ModTime()) {
			t.Fatalf("meta %s = %+v", rel, m)
		}
	}
}

func TestOpen_SPEC0003_R9_loadedLine(t *testing.T) {
	root := tree(t, map[string]string{"a.md": "a", "b.md": "b", "big.md": strings.Repeat("x", note.MaxNoteSize+1)})
	var buf bytes.Buffer
	if _, _, err := note.Open(root, logger(&buf)); err != nil {
		t.Fatal(err)
	}
	lines := find(records(t, &buf), "notes loaded")
	if len(lines) != 1 {
		t.Fatalf("notes loaded lines: %v", lines)
	}
	l := lines[0]
	if l["level"] != "INFO" || l["notes"] != 2.0 || l["skipped"] != 1.0 || l["duration_ms"] == nil {
		t.Fatalf("line %v", l)
	}
	for k := range l {
		switch k {
		case "time", "level", "msg", "notes", "skipped", "duration_ms":
		default:
			t.Fatalf("unexpected field %q in %v", k, l)
		}
	}
}
