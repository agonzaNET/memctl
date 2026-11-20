package gitlog_test

import (
	"testing"

	"github.com/agonzaNET/memctl/internal/gitlog"
)

// A repository without user.name or user.email gives one WARN at start that
// names the missing keys and the identity used instead (SPEC-0008 R17).
func TestCommitter_SPEC0008_R17_identityWarning(t *testing.T) {
	ann := gitlog.Author{Name: "Ann", Email: "ann@example.org"}
	for _, c := range []struct {
		name, userName, userEmail string
		missing                   string // "" for no line
	}{
		{"name only", "Person", "", "user.email"},
		{"email only", "", "person@example.org", "user.name"},
		{"neither", "", "", "user.name, user.email"},
		{"both", "Person", "person@example.org", ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			root := newRepo(t)
			if c.userName != "" {
				git(t, root, "config", "user.name", c.userName)
			}
			if c.userEmail != "" {
				git(t, root, "config", "user.email", c.userEmail)
			}
			h := startCommitter(t, root, ann)
			l := h.log.lines(t, "incomplete git identity")
			if c.missing == "" {
				if len(l) != 0 {
					t.Fatalf("want no line, got %v", l)
				}
				return
			}
			if len(l) != 1 || l[0]["level"] != "WARN" || l[0]["missing"] != c.missing || l[0]["author"] != "Ann <ann@example.org>" {
				t.Fatalf("want one WARN with missing %q, got %v", c.missing, l)
			}
		})
	}
	t.Run("outside git", func(t *testing.T) {
		isolateGit(t)
		h := startCommitter(t, t.TempDir(), ann)
		if l := h.log.lines(t, "incomplete git identity"); len(l) != 0 {
			t.Fatalf("want no line outside git, got %v", l)
		}
	})
}
