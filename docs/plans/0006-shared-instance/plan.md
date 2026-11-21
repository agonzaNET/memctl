# Stage plan 0006 — Shared instance (v0.3.0)

Stage: one `memctl` process per notes root serves several MCP clients on one
machine, such as Claude Desktop's chat and its Cowork and Code sessions
(design D12, D21; ADR-0021). Specs, Accepted on 2026-11-21: SPEC-0009 (new,
R1–R16), and the amendments of SPEC-0006 (R5, acceptance of R1, R3, R4),
SPEC-0004 (R6, R12) and SPEC-0008 (R12). No spec is deferred.

Exit: every new or amended requirement has a passing test that names it, and a
live check by the maintainer shows the notes in a Claude Desktop chat and a
Cowork session at the same time; then v0.3.0 is released.

## Slices

| Slice | Tasks | Ends with |
|---|---|---|
| A — First shared session | T1–T3 | Two `memctl mcp` processes on one root: a note written through the second is read through the first |
| B — Refusal and retry | T4–T5 | A forwarder with another git author is refused; a forwarder retries, takes over a freed lock, or stops with the lock message after 10 s |
| C — Lifetime | T6–T8 | The owner outlives its own client while forwarders remain; signals, a lost owner and a closed stderr behave as specified |
| D — Sessions in the log | T9 | Every `tool call` line names its session; forwarded sessions are logged when they open and close |
| E — Concurrency | T10 | Concurrent edits from two sessions give one success and one conflict, under the race detector |
| F — Close | T11, T14, T12 | README and CHANGELOG; live check in Claude Desktop; specs `Implemented` |
| G — Release | T13 | v0.3.0 tagged and released |

## File map

| File | Change | Why |
|---|---|---|
| `internal/share/doc.go` | create | Package `share`: the socket, the handshake and the byte pipe between a forwarder and the owner, stdlib only (ADR-0002, ADR-0010) |
| `internal/share/hello.go` | create | Handshake lines: the forwarder's hello and the owner's answer, encode, decode, validate, compare (SPEC-0009 R6–R8) |
| `internal/share/listen.go` | create | Owner side: socket path, stale-file removal, listen, mode 0600, too-long path check, accept with the handshake and its 5 s deadline (SPEC-0009 R1, R2, R7) |
| `internal/share/dial.go` | create | Forwarder side: connect, send the hello, read the answer (SPEC-0009 R5–R7) |
| `internal/share/pipe.go` | create | Byte copy in both directions, half-close on stdin EOF, owner-loss detection (SPEC-0009 R9–R11) |
| `internal/share/*_test.go` | create | Tests named `SPEC0009_Rn` for the package |
| `internal/mcpserver/server.go` | modify | Several sessions on one server: `Serve` for stdio keeps its signature; `ServeConn` serves one forwarded connection; a session name per `*mcp.ServerSession` (SPEC-0009 R12, R13) |
| `internal/mcpserver/log.go` | modify | `session` field on the `tool call` line (SPEC-0004 R12, SPEC-0009 R13) |
| `internal/mcpserver/server_test.go` | modify | Two in-memory sessions on one server; the `session` field |
| `cmd/memctl/mcp.go` | modify | Role choice after the lock attempt; owner start-up (listen after load, `sharing` on `started`); hand-off to the forwarder (SPEC-0009 R1–R5) |
| `cmd/memctl/forward.go` | create | The forwarder: retry loop with lock attempts, 10 s limit (a package variable tests shorten), `forwarding`, `stopped` and error lines, exit statuses (SPEC-0009 R4, R5, R9–R11; SPEC-0006 R5) |
| `cmd/memctl/sessions.go` | create | The owner's session set: names, `session opened` / `session closed`, "last session ended" signal (SPEC-0009 R13, R14) |
| `cmd/memctl/lifecycle.go` | modify | Shutdown on the last session or a signal; SIGPIPE ignored so a closed pipe gives an error (SPEC-0004 R6, SPEC-0009 R14–R16) |
| `internal/note/lock.go` | modify | The refusal message gains `, and it did not accept a shared session within 10 s` (SPEC-0006 R5) |
| `cmd/memctl/lock_test.go` | modify | Refusal tests hold the lock from the test process (no socket) and expect the new message; the R1 and R3/R4 tests expect forwarding |
| `cmd/memctl/share_e2e_test.go` | create | Multi-process tests: owner and forwarders with SDK clients, kills, signals, socket file (SPEC-0009, SPEC-0004 R6, SPEC-0008 R12) |
| `cmd/memctl/shutdown_test.go` | modify | Stdin EOF with a forwarded session open (SPEC-0004 R6) |
| `README.md` | modify (T11) | "Several clients on one root" section; the lock message; the socket in `.memory/` |
| `CHANGELOG.md` | modify (T11, T13) | `[Unreleased]`: `Added` shared instance; `Changed` lock message, shutdown on the last session, `session` in the log |
| `docs/design/<date>-v0.3.0-live-check.md`, `docs/design/README.md` | create, modify (T12) | Record of the live check |
| Specs and `docs/specs/README.md` | modify (T12) | Acceptance boxes; `Implemented` |
| `docs/design/<date>-v0.3.0-release.md` | create (T13) | Release record (ADR-0012) |

No new third-party module: `internal/share` uses `net`, `encoding/json`,
`io`, `os` and `time`. The go-sdk stays confined to `internal/mcpserver`
(ADR-0010): `ServeConn` takes an `io.ReadWriteCloser`, not an SDK type.

Interface sketch (inside the module only):

```go
// internal/share
type Hello struct { Memctl string; Protocol int; Version string; PID int; GitAuthor string }
type Answer struct { OK bool; PID int; Version, Error string }
func SocketPath(root string) string
func Listen(root string) (*Listener, error)              // R1, R2: ErrPathTooLong or other error → sharing off
func (l *Listener) Accept(check func(Hello) error) (net.Conn, Hello, error) // R7
func Dial(root string, h Hello) (net.Conn, Answer, error) // R5–R7
func Pipe(stdin io.Reader, stdout io.Writer, conn net.Conn) (ownerLost bool, err error) // R9–R11

// internal/mcpserver
func (s *Server) Serve(ctx context.Context, in io.ReadCloser, out io.WriteCloser) error // session "stdio"
func (s *Server) ServeConn(ctx context.Context, c io.ReadWriteCloser, session string) error
```

## Dependency order

```
T1 → T2 → T3                  (slice A)
T3 → T4, T5                   (slice B; independent of each other)
T3 → T6 → T7 → T8             (slice C)
T2 → T9                       (slice D)
T3 → T10                      (slice E)
all → T11 → T12 → T13         (slices F, G)
```

## Tasks

Each task: write the tests named after the requirements (`…_SPEC0009_R4_…`),
run them and see them fail for the expected reason, commit them
(`test(<pkg>): failing test … (spec: SPEC-NNNN)`), implement, `make verify`,
commit (`feat(<pkg>): …` or `fix(<pkg>): … (spec: SPEC-NNNN)`). Multi-process
tests build the binary once per package run and use a root under `/tmp`
(`tempRoot`): the folder of `t.TempDir()` can make the socket path too long to
share. `make verify` runs without the race detector; `go test -race` is run
by hand on the packages this stage touches (see Verification notes).

### Slice A — First shared session

- [x] **T1 — Handshake lines.** `Hello` and `Answer` with the JSON names of
  SPEC-0009 R6; encode as one line ending in `\n`; decode at most 4 KiB, one
  line; validate `memctl` `"forward"` and `protocol` `1`; the
  `git author differs: …` message; a version comparison that callers turn into
  the `WARN`. Table tests: valid hello, extra fields ignored, missing fields,
  other protocol, no newline within 4 KiB, other author, other version.
  *SPEC-0009 R6, R7, R8 (codec part).*
- [x] **T2 — Owner listens and serves sessions.** `share.Listen`: remove a file at
  the path, listen, `chmod` 0600; a path longer than the platform limit
  (computed from `syscall.RawSockaddrUnix`) returns `ErrPathTooLong` without
  touching the file system. `Accept` sets a 5 s read deadline for the hello,
  answers, clears the deadline. `mcpserver`: `Serve` and `ServeConn` connect
  sessions to one `*mcp.Server` (`Connect`, not `Run`), each with its name.
  `runMCP` (owner path): listen after the load and before `started`; `started`
  gains `sharing`; on a listen error one `WARN` `sharing is off` with `reason`
  and `socket`, and go on. Tests: socket exists with mode 0600 after
  `started`; a stale socket file is replaced; a connection made during a slow
  load (the `afterLock` hook) gets no answer before the load ends; a long root
  path gives the `WARN` and `sharing` `false`, and tools work over stdio.
  *SPEC-0009 R1, R2, R3, R12 (owner part).*
- [x] **T3 — Forwarder.** `runMCP`: when `AcquireLock` returns `*HeldError`,
  run the forwarder instead of exiting: `share.Dial` with this process's
  version, PID and git author; on `ok`, log `forwarding` (`owner_pid`,
  `owner_version`) and `share.Pipe` stdin, stdout and the connection. The
  forwarder never opens the note service or the committer. End-to-end test:
  owner and forwarder on one root, an SDK client on each; both list the tools;
  `note_write` through the forwarder, `note_read` of the same note through the
  owner; the forwarder logs no `notes loaded`; every byte of its stdout
  parses as protocol. Adapt `lock_test.go`: the R1 test expects the second
  process to forward once the first listens; the R3/R4 test expects no lock
  taken and no notes loaded by the second.
  *SPEC-0009 R4, R9, R12; SPEC-0006 R1, R3, R4 (acceptance).*

### Slice B — Refusal and retry

- [x] **T4 — Refusals and version warning.** Owner: `Accept`'s check refuses
  another author with the R7 message; a malformed line, `protocol` 2 or
  silence for 5 s gets `ok: false` and the connection is closed (raw-socket
  tests). Forwarder: on `ok: false`, one `ERROR` with the owner's message,
  status 1, no retry (the test counts connection attempts on a fake
  listener). A different version: one `WARN` `version differs` on each side,
  and tool calls work (the forwarder's version is set through a test hook
  in-process).
  *SPEC-0009 R7, R8.*
- [x] **T5 — Retry, take-over and timeout.** Forwarder loop every 100 ms for at
  most `forwardTimeout` (10 s; a package variable that in-process tests set to
  1 s): try the lock — on success continue as owner from the load on; else
  dial. On timeout, `HeldError` with the new suffix (in `lock.go`) through the
  existing R5 path (terminal rule, given and resolved root). Tests: a
  forwarder started during the owner's slow load connects once it listens; a
  forwarder started while the owner is in its final commit becomes owner; a
  lock held by the test process itself (no socket) gives the SPEC-0006 R5
  message and status 1, once in-process with a 1 s limit and once as a
  spawned process with the real 10 s (`t.Parallel`); the symlinked-root and
  empty-PID cases of the existing R5 tests with the new message.
  *SPEC-0009 R5; SPEC-0006 R5.*

### Slice C — Lifetime

- [x] **T6 — Forwarder exits.** `Pipe`: on stdin EOF `CloseWrite`, copy to
  stdout until the connection ends, then `stopped` and status 0; on SIGINT or
  SIGTERM close the connection and status 0; when the connection ends while
  stdin is open, `ERROR` `memctl (pid <N>) serving the root stopped` and
  status 1. Tests for the three cases, the last with SIGKILL to the owner,
  followed by a new `memctl mcp` on the root that becomes the owner.
  *SPEC-0009 R10, R11.*
- [x] **T7 — Owner lifetime.** `sessions.go`: the set of open sessions
  (`stdio` plus forwarded), a channel closed when the set becomes empty after
  the own session ended. Stdin EOF ends only the own session; the shutdown
  context is cancelled when the set is empty; then the listener is closed (the
  socket file goes with it), connections without an `ok` answer are closed,
  the final commit runs, status 0. Tests: owner's stdin closed while a
  forwarder is connected — the forwarder's calls still work; after the
  forwarder's stdin closes, the final commit exists with the owner's author,
  the socket file is gone, status 0; with sharing off, stdin EOF stops the
  process as before.
  *SPEC-0009 R14; SPEC-0004 R6; SPEC-0008 R12 (shared).*
- [x] **T8 — Signals and closed pipes.** SIGINT or SIGTERM: stop accepting,
  let the call in progress finish, close every forwarded connection, final
  commit, exit (second signal: SPEC-0008 R13, unchanged). `signal.Ignore`
  for SIGPIPE in `runMCP`, so a write to a closed stdout or stderr returns
  an error. Tests: SIGTERM to the owner during a slow call of a forwarded
  session (the `beforeCall` hook) — the call completes, the forwarder exits
  with status 1, the commit exists, the owner exits with status 0; the
  owner's stderr pipe closed by the test, then a tool call through a
  forwarder succeeds and the owner keeps running.
  *SPEC-0009 R15, R16.*

### Slice D — Sessions in the log

- [x] **T9 — `session` field.** `handle` looks up the session name of the
  call's `*mcp.ServerSession` and `callLog.write` adds `session`; `stdio` for
  the own session, `forward-<n>` by order of the `ok` answer. The owner logs
  `session opened` and `session closed` with `session` and `forwarder_pid`.
  Tests: a captured owner log of one call per session shows `stdio` and
  `forward-1`, the two session lines with the forwarder's PID; no line holds
  note content (the existing content scan runs over the new lines).
  *SPEC-0009 R13; SPEC-0004 R12.*

### Slice E — Concurrency

- [x] **T10 — Two sessions, one service.** Two SDK clients (stdio and
  forwarded) call `note_edit` on one note with the same `expected_sha` at the
  same moment, twenty rounds: one success and one conflict per round; every
  commit has the owner's author; no data race (`-race`).
  *SPEC-0009 R12.*

### Slice F — Close

- [ ] **T14 — Release-audit fixes.** From `RELEASE-AUDIT.md` of 2026-11-21:
  W1 — `share.Dial` waits for the answer no later than the forwarder's
  deadline (test with an owner that accepts and never answers); W2 — checked,
  no code change: the Go runtime starts child processes with the default
  action for a signal the program ignores, so `git` and its hooks already
  get default SIGPIPE; R17 and a guard test (a `pre-commit` hook reports its
  `SigIgn` mask) keep it so; W3 — the second
  session-name lookup under the mutex (staticcheck SA2001); W5 — `CLAUDE.md`
  describes the shared instance; N1 — comment order in `runMCP`; N2 — the
  test helper returns its error last (ST1008).
  *SPEC-0009 R5, R16, R17.*
- [x] **T11 — README and changelog.** README: a section "Several clients on one
  root" (Claude Desktop chat and Cowork, Claude Code and Desktop; the owner
  and forwarders; the socket in `.memory/`; what happens when the owner
  stops; a root too long to share), and the new lock message.
  `CHANGELOG.md` `[Unreleased]`: `Added` — several MCP clients share one notes
  root through one process; `Changed` — the lock message, the process stops
  when its last client session ends, the `tool call` log line carries
  `session`.
  *No requirement; ADR-0007.*
- [ ] **T12 — Live check and close.** `make build GOOS=darwin GOARCH=arm64`.
  The maintainer installs the binary as `~/.local/bin/memctl` and restarts
  Claude Desktop with the existing `infradocs` entry, then:
  1. `search` in a chat and in a Cowork session, at the same time: both
     return hits;
  2. the Desktop log `mcp-server-infradocs.log` shows `sharing` `true` on
     `started`, a `forwarding` line, `session` on the `tool call` lines, and
     no `Couldn't start for Cowork and Code sessions`;
  3. a `note_write` on a scratch note in Cowork, read back in the chat, and a
     single commit for it with the owner's author (then `note_delete`);
  4. quit and reopen Desktop: both surfaces work again.

  Write `docs/design/<date>-v0.3.0-live-check.md` and list it in
  `docs/design/README.md`; tick every acceptance box of SPEC-0009, SPEC-0006,
  SPEC-0004 and SPEC-0008; set the four specs and their index rows to
  `Implemented`; tick this plan. Commit
  `docs: shared instance implemented (spec: SPEC-0009, SPEC-0006, SPEC-0004, SPEC-0008)`.
  *SPEC-0009 live check.*

### Slice G — Release

- [ ] **T13 — Release 0.3.0.** The steps of `docs/agents/environment.md`
  "Releasing": rename `[Unreleased]` to `[0.3.0] - <date>`, open a fresh
  `[Unreleased]` with the comment, update the link references (compare
  `v0.2.0...v0.3.0`), `make verify`, commit, `git tag -a v0.3.0 -m "memctl
  0.3.0"`. The maintainer pushes and runs
  `gh workflow run release.yml -f tag=v0.3.0`. Verify the downloaded assets
  against `SHA256SUMS`, run one on a fresh root, write
  `docs/design/<date>-v0.3.0-release.md` and list it in the design README.
  *ADR-0012; no requirement.*

## Requirement coverage

| Spec | Requirements | Tasks |
|---|---|---|
| SPEC-0009 | R1, R2, R3 | T2 |
| SPEC-0009 | R4, R9 | T3 |
| SPEC-0009 | R5 | T5 |
| SPEC-0009 | R6 | T1, T3 |
| SPEC-0009 | R7, R8 | T1, T4 |
| SPEC-0009 | R10, R11 | T6 |
| SPEC-0009 | R12 | T2, T3, T10 |
| SPEC-0009 | R13 | T9 |
| SPEC-0009 | R14 | T7 |
| SPEC-0009 | R15, R16 | T8 |
| SPEC-0009 | R5 (answer wait), R16, R17 | T14 |
| SPEC-0009 | Live check | T12 |
| SPEC-0006 | R1, R3, R4 (acceptance) | T3 |
| SPEC-0006 | R5 | T5 |
| SPEC-0004 | R6 | T7 |
| SPEC-0004 | R12 | T9 |
| SPEC-0008 | R12 (shared) | T7 |

## Definition of done

- `make verify` passes (tests with `-race`, `go vet`, `gofmt`,
  `check-docs.sh`), and `govulncheck ./...` reports no vulnerability the code
  calls.
- Every new or amended requirement has at least one passing test whose name
  carries it; every acceptance box of SPEC-0009, SPEC-0006, SPEC-0004 and
  SPEC-0008 is ticked; the four specs and their index rows say `Implemented`.
- The import-scan test (ADR-0010) passes: `internal/share` imports no
  third-party module, and the go-sdk is imported by `internal/mcpserver` only.
- The live check is recorded under `docs/design/` and listed in its README.
- `CHANGELOG.md` holds the `[0.3.0]` section, and the release record is
  written.

## Verification notes

Recorded after T1–T11 on the development VM (linux/arm64, Go 1.26.11):

- `make verify` passes; `govulncheck ./...`: no vulnerabilities found.
- `CGO_ENABLED=1 go test -race` on `cmd/memctl`, `internal/mcpserver`,
  `internal/share` and `internal/note`: clean.
- T10's tests passed when first written: the behaviour comes from the note
  service mutex, which every session shares; they stay as guards.
- SPEC-0009 R15 ("SIGTERM during a slow call of a forwarded session") is shown
  in two tests: `TestServer_SPEC0009_R15_forwardedCallFinishes`
  (`internal/mcpserver`, a forwarded session returns only after its call in
  progress) and `TestE2E_SPEC0009_R15_SIGTERMToOwner` (forwarder status 1,
  final commit, owner status 0); the spawned binary has no hook to slow a
  call.
- SPEC-0009 R5's take-over is shown with a lock held by the test process and
  released during the retry (`TestCLI_SPEC0009_R5_takeOverFreedLock`), which
  is the state a forwarder sees while an owner exits.
- The disk of the VM's session folder filled up during T8; `GOCACHE`,
  `GOTMPDIR` and `TMPDIR` moved to `/tmp`. No effect on the code.

