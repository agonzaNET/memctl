package gitlog_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/agonzaNET/memctl/internal/gitlog"
)

func TestIsRepo_SPEC0005_R7(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	repo := t.TempDir()
	if out, err := exec.Command("git", "init", "-q", repo).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, out)
	}
	sub := filepath.Join(repo, "notes", "deep")
	os.MkdirAll(sub, 0o755)
	outside := t.TempDir()
	ctx := context.Background()
	for _, c := range []struct {
		root string
		want bool
	}{{repo, true}, {sub, true}, {outside, false}} {
		got, err := gitlog.IsRepo(ctx, c.root)
		if err != nil || got != c.want {
			t.Errorf("IsRepo(%s) = %v, %v; want %v", c.root, got, err, c.want)
		}
	}

	t.Setenv("PATH", t.TempDir())
	if _, err := gitlog.IsRepo(ctx, repo); !errors.Is(err, gitlog.ErrGitMissing) {
		t.Fatalf("without git: %v", err)
	}
}
