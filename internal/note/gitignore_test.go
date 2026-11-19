package note_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/agonzaNET/memctl/internal/note"
)

// ignoreCase is one path to check, with git's answer when git is not
// installed.
type ignoreCase struct {
	rel   string
	isDir bool
	want  bool
}

func TestGitignore_SPEC0003_R5_patterns(t *testing.T) {
	files := map[string]string{
		".gitignore": strings.Join([]string{
			"# comment",
			"",
			"*.log",
			"drafts/",
			"/anchored.md",
			"docs/*.tmp",
			"**/cache",
			"build/**",
			"a/**/z.md",
			"report-[0-9].md",
			"?.bak",
			"secret*",
			"!secret-ok.md",
			`\#hash.md`,
		}, "\n") + "\n",
		"sub/.gitignore":          "!keep.log\nlocal.md\n/only-here.md\n",
		"x.log":                   "",
		"sub/keep.log":            "",
		"sub/other.log":           "",
		"drafts/d.md":             "",
		"nested/drafts/e.md":      "",
		"anchored.md":             "",
		"nested/anchored.md":      "",
		"docs/x.tmp":              "",
		"docs/deep/y.tmp":         "",
		"p/cache/c.md":            "",
		"build/out/b.md":          "",
		"a/z.md":                  "",
		"a/b/c/z.md":              "",
		"report-1.md":             "",
		"report-x.md":             "",
		"x.bak":                   "",
		"xy.bak":                  "",
		"secret.md":               "",
		"secret-ok.md":            "",
		"#hash.md":                "",
		"sub/local.md":            "",
		"local.md":                "",
		"sub/only-here.md":        "",
		"sub/deeper/only-here.md": "",
		"plain.md":                "",
	}
	root := tree(t, files)
	cases := []ignoreCase{
		{"x.log", false, true},
		{"sub/keep.log", false, false},
		{"sub/other.log", false, true},
		{"drafts", true, true},
		{"nested/drafts", true, true},
		{"anchored.md", false, true},
		{"nested/anchored.md", false, false},
		{"docs/x.tmp", false, true},
		{"docs/deep/y.tmp", false, false},
		{"p/cache", true, true},
		{"build/out/b.md", false, true},
		{"a/z.md", false, true},
		{"a/b/c/z.md", false, true},
		{"report-1.md", false, true},
		{"report-x.md", false, false},
		{"x.bak", false, true},
		{"xy.bak", false, false},
		{"secret.md", false, true},
		{"secret-ok.md", false, false},
		{"#hash.md", false, true},
		{"sub/local.md", false, true},
		{"local.md", false, false},
		{"sub/only-here.md", false, true},
		{"sub/deeper/only-here.md", false, false},
		{"plain.md", false, false},
	}
	gitAnswers := gitCheckIgnore(t, root, cases)
	for _, c := range cases {
		want := c.want
		if gitAnswers != nil {
			want = gitAnswers[c.rel]
			if want != c.want {
				t.Errorf("table disagrees with git for %s: git says %v", c.rel, want)
			}
		}
		if got := note.Ignored(root, c.rel, c.isDir); got != want {
			t.Errorf("%s: ignored=%v, want %v", c.rel, got, want)
		}
	}
}

// gitCheckIgnore asks git for each case, in a repository at root with no
// user or system configuration. It returns nil when git is not installed.
func gitCheckIgnore(t *testing.T, root string, cases []ignoreCase) map[string]bool {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		return nil
	}
	env := append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1")
	init := exec.Command("git", "init", "-q", root)
	init.Env = env
	if out, err := init.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, out)
	}
	out := map[string]bool{}
	for _, c := range cases {
		p := c.rel
		if c.isDir {
			p += "/"
		}
		cmd := exec.Command("git", "-C", root, "check-ignore", "-q", p)
		cmd.Env = env
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		err := cmd.Run()
		if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() == 1 {
			out[c.rel] = false
		} else if err == nil {
			out[c.rel] = true
		} else {
			t.Fatalf("git check-ignore %s: %v %s", p, err, stderr.String())
		}
	}
	return out
}

func TestGitignore_SPEC0003_R5_R11_walkAndReload(t *testing.T) {
	root := tree(t, map[string]string{
		"a.md":         "a",
		"drafts/d.md":  "d",
		"x/ignored.md": "i",
		"x/.gitignore": "ignored.md\n",
	})
	var buf bytes.Buffer
	s, _, err := note.Open(root, logger(&buf))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(s.Index().Paths(), []string{"a.md", "drafts/d.md"}) {
		t.Fatalf("paths %v", s.Index().Paths())
	}
	write(t, root, ".gitignore", "drafts/\n")
	check(t, s)
	if !reflect.DeepEqual(s.Index().Paths(), []string{"a.md"}) {
		t.Fatalf("after ignoring drafts: %v", s.Index().Paths())
	}
	os.Remove(filepath.Join(root, ".gitignore"))
	check(t, s)
	if !reflect.DeepEqual(s.Index().Paths(), []string{"a.md", "drafts/d.md"}) {
		t.Fatalf("after removing the rule: %v", s.Index().Paths())
	}
}
