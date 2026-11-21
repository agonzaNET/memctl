# Release Audit — memctl

**Verdict: GO** ✅ (conditional on the open live check)
No blockers: build, vet, format, module checks and the full test suite (also under `-race`) pass; 6 warnings, 2 nits. The release itself must still wait for the maintainer's live check (stage plan 0006, T12).

_Audited 2026-11-21 (UTC) against `main` @ `f14f10c`; last release tag `v0.2.0`._
_Scope: full-repo read-only sweep, with the 0.3.0 changes (shared instance, 40 files since `v0.2.0`) reviewed line by line. No files were modified._

## Checks run

| Check | Command | Result |
|---|---|---|
| Build | `go build ./...` | ✅ pass |
| Vet | `go vet ./...` | ✅ pass |
| Format | `gofmt -l .` | ✅ no files |
| Lint | `staticcheck ./...` (v0.8.2, via `go run`) | ⚠️ 2 findings (W3, N2) |
| Modules | `go mod verify`; `go mod tidy -diff` | ✅ verified; tidy |
| Tests | `go test -race -count=1 ./...` (CGO on) | ✅ all 8 packages pass |
| Project gate | `make verify` (tests, vet, fmt-check, `check-docs.sh`, script tests) | ✅ pass |
| Vulnerabilities | `govulncheck ./...` (run earlier the same day, same code) | ✅ none found |
| Debug leftovers | grep for TODO/FIXME/XXX/`fmt.Print`/`t.Skip`/`panic` in changed Go files | ✅ none (the one `fmt.Fprintln` is the intended terminal line of SPEC-0006 R5) |

## 🟡 Warnings (should review)

- **W1** `internal/share/dial.go:26` — one attempt waits up to 10 s (`answerTimeout`) for the owner's answer, so against an owner that accepts but never answers, a forwarder can take about 20 s before it stops. SPEC-0009 R5 says "for at most 10 s after it started". The answer deadline should be capped at the time left before the forwarder's deadline.
- **W2** `cmd/memctl/lifecycle.go` (`newShutdown`) — `signal.Ignore(syscall.SIGPIPE)` sets SIGPIPE to "ignored" at the OS level, which child processes inherit: every `git` call and the repository's git hooks then run with SIGPIPE ignored. `signal.Notify` on a drained channel gives memctl the same behaviour (a write to a closed fd 1 or 2 returns `EPIPE`, SPEC-0009 R16) while children start with the default action.
- **W3** `internal/mcpserver/server.go:103` — staticcheck SA2001 "empty critical section". The `Lock(); Unlock()` is a deliberate wait for a connecting session, but it reads as a bug; do the second `Load` while holding the mutex.
- **W4** Release gate — SPEC-0009, SPEC-0006, SPEC-0004 and SPEC-0008 are `Accepted`, not `Implemented`; the SPEC-0009 live-check box and plan tasks T12 and T13 are open. Expected at this point; the release waits for T12.
- **W5** `CLAUDE.md` "What this project is" — still describes one process per root with no word on the shared instance (the forwarder, the socket in `.memory/`); the layout block was updated, the description was not. Agent guide drift.
- **W6** macOS is not exercised by any test: the suite ran on linux/arm64, and darwin/arm64 is only cross-built. The socket length limit (103 bytes there), `flock`, signals and SIGPIPE behaviour on macOS are first seen in the live check.

## ⚪ Nits (optional)

- **N1** `cmd/memctl/mcp.go` (after `srv.Serve`) — the older comment "Serve has returned … commit what is queued at once" now sits above the new `closeWhenIdle` block it does not describe; reorder.
- **N2** `internal/share/greet_test.go:14` — staticcheck ST1008: test helper `greet` returns the error first.

## Notes

- Working tree was clean at the start of the audit; this report is the only new file (untracked).
- The VM's session disk filled once during the stage; build caches live under `/tmp`. No effect on the code or results.
- The darwin/arm64 live-check binary (`bin/memctl`, git-ignored) was built from `8107d55`; docs-only commits follow it, so it matches the audited code.
