package main

import (
	"strings"
	"testing"
)

func TestCLI_SPEC0008_gitAuthor(t *testing.T) {
	root := notesRoot(t)
	r := runBin(t, binVersioned, nil, "mcp", "--root", root, "--git-author", "no-email-zzz")
	if r.code != 2 || !strings.Contains(r.stderr, "--git-author") || strings.Contains(r.stderr, "no-email-zzz") {
		t.Fatalf("bad flag: %+v", r)
	}
	r = runBin(t, binVersioned, []string{"MEMCTL_GIT_AUTHOR=Ann <a@b@c>"}, "mcp", "--root", root)
	if r.code != 2 || !strings.Contains(r.stderr, "MEMCTL_GIT_AUTHOR") || strings.Contains(r.stderr, "a@b@c") {
		t.Fatalf("bad env: %+v", r)
	}
	for _, c := range []struct {
		env  []string
		args []string
		want string
	}{
		{nil, nil, "memctl <memctl@localhost>"},
		{[]string{"MEMCTL_GIT_AUTHOR=Env Person <env@example.org>"}, nil, "Env Person <env@example.org>"},
		{[]string{"MEMCTL_GIT_AUTHOR=Env Person <env@example.org>"}, []string{"--git-author", "Ann <ann@example.org>"}, "Ann <ann@example.org>"},
	} {
		args := append([]string{"mcp", "--root", root}, c.args...)
		r := runBin(t, binVersioned, c.env, args...)
		started := byMsg(jsonLines(t, r.stderr), "started")
		if r.code != 0 || started == nil || started["git_author"] != c.want {
			t.Fatalf("%v %v: started %v, want git_author %q; %s", c.env, c.args, started, c.want, r.stderr)
		}
	}
}
