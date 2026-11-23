package main

import (
	"context"
	"strings"
	"testing"
)

func TestE2E_SPEC0011_R10_R12_ownersScopeAndWarning(t *testing.T) {
	for _, fwdScope := range []string{"", "work notes"} {
		t.Run("forwarder scope "+fwdScope, func(t *testing.T) {
			root := gitRepo(t)
			owner := connectArgs(t, root, "--scope", "personal notes")
			waitFor(t, owner.stderr, `"msg":"started"`)
			fwd := connectArgs(t, root, "--scope", fwdScope)
			waitFor(t, fwd.stderr, `"msg":"forwarding"`)

			// R10: the forwarded session is served with the owner's scope.
			if got := fwd.cs.InitializeResult().Instructions; !strings.HasSuffix(got, scopeLine+"personal notes") {
				t.Fatalf("forwarded instructions do not end with the owner's scope line: %q", got)
			}
			res, err := fwd.cs.ListTools(context.Background(), nil)
			if err != nil || len(res.Tools) != 8 {
				t.Fatalf("tools: %v", err)
			}
			for _, tool := range res.Tools {
				if !strings.HasSuffix(tool.Description, scopeLine+"personal notes") {
					t.Errorf("forwarded %s description does not end with the owner's scope line", tool.Name)
				}
			}

			// R12: one WARN on each side, and the session goes on.
			waitFor(t, owner.stderr, `"msg":"scope differs"`)
			fwd.write(t, "Written through the forwarder")
			ol, fl := jsonLines(t, owner.stderr.String()), jsonLines(t, fwd.stderr.String())
			if countMsg(ol, "scope differs") != 1 || countMsg(fl, "scope differs") != 1 {
				t.Fatalf("scope differs lines: owner %d, forwarder %d", countMsg(ol, "scope differs"), countMsg(fl, "scope differs"))
			}
			ow := byMsg(ol, "scope differs")
			if ow["level"] != "WARN" || ow["forwarder_pid"] != float64(fwd.cmd.Process.Pid) || ow["forwarder_scope"] != fwdScope {
				t.Fatalf("owner WARN %v", ow)
			}
			fw := byMsg(fl, "scope differs")
			if fw["level"] != "WARN" || fw["owner_scope"] != "personal notes" {
				t.Fatalf("forwarder WARN %v", fw)
			}
		})
	}
}

func TestE2E_SPEC0011_R12_equalScopesLogNothing(t *testing.T) {
	root := gitRepo(t)
	owner := connectArgs(t, root, "--scope", "personal notes")
	waitFor(t, owner.stderr, `"msg":"started"`)
	fwd := connectArgs(t, root, "--scope", " personal notes ")
	waitFor(t, fwd.stderr, `"msg":"forwarding"`)
	fwd.write(t, "Written through the forwarder")
	waitFor(t, owner.stderr, `"msg":"tool call"`)
	if strings.Contains(owner.stderr.String(), "scope differs") || strings.Contains(fwd.stderr.String(), "scope differs") {
		t.Fatalf("equal scopes logged a difference:\n%s\n%s", owner.stderr.String(), fwd.stderr.String())
	}
}
