# SPEC-0009 - Shared instance: one owner per notes root, forwarders over a Unix socket

## Status

Implemented

> Amended on 2026-11-23 by [SPEC-0011](../0011-instance-scope/spec.md)
> (ADR-0025): the hello and the `ok` answer (R6) carry an optional `scope`;
> owner and forwarder each log `WARN` `scope differs` when the scopes differ,
> beside `version differs` (R8); every session gets the owner's scope.

<!--
Status values: Draft | Accepted | Implemented | Superseded by SPEC-NNNN | Deprecated
Lifecycle: Draft (under review) → Accepted (approved; the code may be built,
but at least one acceptance criterion, for example a live test, is still
open) → Implemented (shipped; code satisfies this spec and all acceptance
criteria are ticked). Superseded by SPEC-NNNN and Deprecated are terminal.
-->

> Amended on 2026-11-21 after the release audit (W2): R17 added; child
> processes keep the default SIGPIPE action.
>
> Amended on 2026-11-22 after the v0.3.0 release workflow failed: R17 covers
> the `git` processes memctl starts, not the hooks. git 2.54 and later runs
> hooks with SIGPIPE ignored, so a hook's setting comes from git, not from
> memctl. The old test checked a hook and passed only because git before 2.54
> resets SIGPIPE at start; it hid that memctl ignored SIGPIPE for itself.

**Governing ADRs:** ADR-0002, ADR-0005, ADR-0014, ADR-0016, ADR-0021

## Purpose

Let several MCP clients on one machine use the same notes root at the same
time, such as Claude Desktop's chat and its Cowork and Code sessions, which
start one `memctl mcp` each from a single configuration entry. The process
that holds the instance lock (the owner) serves every client; any later
`memctl mcp` on that root (a forwarder) passes its client's bytes to the owner
over a Unix socket (design D12, ADR-0021).

## Scope

**In scope**

- `cmd/memctl`: choosing the owner or forwarder role after the lock attempt,
  the forwarder's retry, handshake and byte copy, and the owner's lifetime
  across several sessions.
- The owner's socket listener, handshake and per-connection MCP sessions
  (`internal/mcpserver` and a new package or file named after what it does,
  ADR-0002).
- The `session` field of the `tool call` log line.

**Out of scope**

- The lock itself (SPEC-0006), the tools (SPEC-0004, SPEC-0007) and the
  committer (SPEC-0008), which this spec reuses unchanged apart from the
  amendments named in those specs.
- HTTP transport, clients on other machines and authentication other than file
  permissions (design backlog after v2).
- A forwarder that takes over its client's session in place when the owner dies
  (ADR-0021, alternatives).
- Windows and file systems that do not support Unix sockets; on those, sharing
  is off (R2).

## Inputs

- The flags and environment variables of `memctl mcp` (SPEC-0004, SPEC-0008).
  No flag or variable is added. A forwarder uses `--root` to find the socket,
  `--git-author` in the handshake and `--log-level` for its own log.
- The result of the instance lock attempt (SPEC-0006 R1, R5).
- The socket `<root>/.memory/memctl.sock`, where `<root>` is the resolved root
  (SPEC-0003 R1).

## Outputs

- **Socket.** `<root>/.memory/memctl.sock`, mode 0600, present while an owner
  with sharing on is running. `.memory/.gitignore` already keeps it out of git.
- **Handshake.** Before any MCP byte, the forwarder sends one line and the owner
  answers with one line, each a JSON object followed by `\n`:

  ```text
  → {"memctl":"forward","protocol":1,"version":"0.3.0","pid":4711,"git_author":"memctl <memctl@localhost>"}
  ← {"ok":true,"pid":4242,"version":"0.3.0"}
  ← {"ok":false,"error":"git author differs: this process has Ann <ann@example.org>; memctl (pid 4242) serving the root has memctl <memctl@localhost>"}
  ```

- **Owner log lines** (JSON on stderr, in addition to SPEC-0004's):
  `started` gains `sharing` (`true` or `false`); `WARN` `sharing is off`
  with `reason` and `socket`; `INFO` `session opened` and `session closed`
  with `session` and, for a forwarded session, `forwarder_pid`; `WARN`
  `version differs` with `forwarder_version`; every `tool call` line gains
  `session`.
- **Forwarder log lines:** `INFO` `forwarding` with `owner_pid` and
  `owner_version`; `WARN` `version differs` with `owner_version`; `ERROR`
  lines of R7, R11 and SPEC-0006 R5; `INFO` `stopped`.
- **Exit status.** Owner: as SPEC-0004 R6 and SPEC-0008 R12, R13. Forwarder: 0
  after stdin EOF or a signal; 1 after a refusal, a timeout or the loss of the
  owner.

## Behaviour & Requirements

### Roles

- **R1** — A `memctl mcp` that takes the instance lock (SPEC-0006 R1) is the
  owner. After the startup load (SPEC-0003 R8) and before the `started` line,
  the owner MUST remove any file at `<root>/.memory/memctl.sock`, listen on a
  Unix socket at that path, and set its mode to 0600. Connections MUST NOT be
  accepted before the load is complete.
- **R2** — When the socket cannot be created (a path longer than the platform
  allows, or any other error), the owner MUST log one `WARN` `sharing is off`
  with the reason and the socket path, and go on serving its own stdio as the
  only session.
- **R3** — The owner's `started` line MUST carry `sharing`: `true` when the
  socket listens, `false` after R2.
- **R4** — A `memctl mcp` that finds the lock held (SPEC-0006 R5) MUST NOT load
  the notes, start a git committer or serve tools itself. It MUST become a
  forwarder: connect to the socket and do the handshake (R6).
- **R5** — When the connection fails, or closes before the owner's answer, the
  forwarder MUST retry every 100 ms, for at most 10 s after it started: first
  try the lock, and continue as owner (R1) when it gets it; otherwise connect
  again. After 10 s without an `ok` answer it MUST log the message of
  SPEC-0006 R5 and exit with status 1.

### Handshake

- **R6** — The forwarder MUST send one line before anything else: a JSON object
  with `memctl` `"forward"`, `protocol` `1`, its `version`, its `pid` and its
  `git_author` (as `Name <email>`), then `\n`. The owner MUST answer with one
  line: `{"ok":true,"pid":<owner pid>,"version":"<owner version>"}`, or
  `{"ok":false,"error":"<message>"}` followed by closing the connection. No MCP
  byte MAY pass in either direction before an `ok` answer.
- **R7** — The owner MUST refuse a first line that is not such an object, names
  another `protocol`, or does not arrive within 5 s of the connection. It MUST
  refuse a `git_author` that differs from its own, with the message
  `git author differs: this process has <forwarder author>; memctl (pid <owner pid>) serving the root has <owner author>`.
  A refused forwarder MUST log one `ERROR` line with the owner's message and
  exit with status 1, without retrying.
- **R8** — When the versions differ, the owner and the forwarder MUST each log
  one `WARN` `version differs` naming the other's version, and go on.

### Forwarding

- **R9** — After an `ok` answer, the forwarder MUST log one `INFO` `forwarding`
  line with the owner's PID and version, then copy its stdin to the connection
  and the connection to its stdout, byte for byte, in both directions at once.
  It MUST write nothing else to stdout (SPEC-0004 R4).
- **R10** — On stdin EOF the forwarder MUST close the write half of the
  connection, keep copying to stdout until the owner closes the connection, log
  `stopped` and exit with status 0. On SIGINT or SIGTERM it MUST close the
  connection and exit with status 0.
- **R11** — When the owner closes the connection while the forwarder's stdin is
  still open, the forwarder MUST log one `ERROR` line
  `memctl (pid <owner pid>) serving the root stopped` and exit with status 1,
  so that its client can start it again (R4, R5 then choose its role anew).

### Owner sessions

- **R12** — The owner MUST serve its own stdio and every connection with an
  `ok` handshake as separate MCP sessions of one MCP server, over one note
  service and one git committer. Tool calls from all sessions MUST be
  serialised by the note service mutex (SPEC-0004 R7), and every tool write
  MUST be committed with the owner's git author (SPEC-0008).
- **R13** — Every `tool call` line MUST carry `session`: `stdio` for the owner's
  own session, `forward-<n>` for a forwarded one, where `<n>` counts accepted
  sessions from 1 in the order of their `ok` answer. The owner MUST log one
  `INFO` `session opened` and one `session closed` line per forwarded session,
  with `session` and `forwarder_pid`.

### Owner lifetime

- **R14** — Stdin EOF MUST end only the owner's own session. The owner MUST keep
  serving while any session is open. When its own session and every forwarded
  session have ended, it MUST start the shutdown of SPEC-0004 R6: close the
  listener so the socket file is removed, close any connection without an `ok`
  answer, make the final commit (SPEC-0008 R12) and exit with status 0. An
  owner with sharing off (R2) therefore stops on stdin EOF, as before.
- **R15** — On SIGINT or SIGTERM the owner MUST start the shutdown at once: stop
  accepting, let the tool call in progress finish, close every forwarded
  connection, make the final commit and exit (SPEC-0004 R6, SPEC-0008 R12,
  R13).
- **R16** — The owner MUST NOT be stopped by a closed stdout or stderr: a write
  to a closed pipe MUST fail with an error instead of ending the process, and
  a log line that cannot be written is dropped.
- **R17** — The `git` processes that memctl starts MUST start with the default
  SIGPIPE action: R16 MUST NOT be met by ignoring SIGPIPE for the whole
  process. The repository's hooks get their SIGPIPE action from `git`.

## Error handling

| Failure | Result |
|---|---|
| Socket path too long, or the listen fails | Owner: `WARN` `sharing is off`, serves its own stdio only (R2) |
| Owner still loading, exiting, or without a socket | Forwarder retries for 10 s, may become owner; then SPEC-0006 R5 message, status 1 (R5) |
| Malformed or late handshake, other protocol | Owner answers `ok: false` and closes; forwarder `ERROR`, status 1 (R7) |
| Different `--git-author` | Same, with the `git author differs` message (R7) |
| Different version | `WARN` on both sides; the session goes on (R8) |
| Owner killed or stopped while a forwarder runs | Forwarder `ERROR`, status 1; the client restarts it (R11) |
| Forwarded connection breaks | Owner ends that session, logs `session closed`; other sessions go on (R13, R14) |

## Acceptance criteria — "Done looks like"

- [x] **R1, R3** — An end-to-end test: the owner's socket exists with mode 0600
  after the `started` line, which carries `sharing` `true`; a stale socket file
  left before start is replaced; a connection attempted during a slow load is
  not answered before the load ends.
- [x] **R2** — A root whose socket path is longer than the platform limit gives
  the `WARN` and `sharing` `false`; tools work over stdio; stdin EOF stops the
  process with status 0.
- [x] **R4, R9, R12** — Two processes on one root, each with an SDK client: both
  list the tools; a note written through the second is read through the
  first; the second process loads no notes (no `notes loaded` line) and its
  stdout carries only protocol bytes.
- [x] **R5** — A forwarder started while the owner is loading connects once the
  socket listens; a forwarder started while the owner exits becomes the owner;
  a lock held by a process without a socket gives the SPEC-0006 R5 message
  and status 1 after 10 s.
- [x] **R6, R7** — A raw connection that sends a malformed line, `protocol` 2,
  nothing for 5 s, or another git author gets `ok: false` with the expected
  message and is closed; the forwarder with another `--git-author` exits with
  status 1 after one `ERROR` line and does not retry.
- [x] **R8** — A forwarder reporting another version: one `WARN` on each side,
  and tool calls work.
- [x] **R10** — Closing the forwarder's stdin gives status 0 and a `stopped`
  line; the owner logs `session closed`; SIGTERM to the forwarder gives
  status 0.
- [x] **R11** — SIGKILL to the owner: the forwarder logs the `ERROR` and exits
  with status 1; a new `memctl mcp` started then becomes the owner.
- [x] **R12** — Concurrent `note_edit` calls from two sessions on one note: one
  succeeds and the other gets the conflict error; every commit has the
  owner's author.
- [x] **R13** — A captured owner log of one call per session shows `session`
  `stdio` and `forward-1`, and the `session opened` and `session closed`
  lines with the forwarder's PID; no line holds note content.
- [x] **R14** — The owner's stdin closed while a forwarder is connected: the
  owner keeps serving the forwarder; after the forwarder's stdin closes, the
  owner makes the final commit, the socket file is gone, and it exits with
  status 0.
- [x] **R15** — SIGTERM to the owner during a slow tool call of a forwarded
  session: the call completes, the forwarder exits with status 1, the final
  commit exists, the owner exits with status 0.
- [x] **R16** — With the owner's stderr pipe closed by the reader, a later tool
  call still succeeds and the process keeps running.
- [x] **R17** — Every `git` that memctl starts for a tool write reports SIGPIPE
  as not ignored (on Linux, bit 13 of `SigIgn` in `/proc/<pid>/status` is
  clear), also when memctl itself was started with SIGPIPE ignored.
- [x] **Live check** — Claude Desktop with one `memctl` entry: a `search` works in
  a chat and in a Cowork session at the same time; the Desktop log shows no
  `Couldn't start for Cowork and Code sessions`; after quitting and reopening
  Desktop, both work again.

## References

- [ADR-0021](../../adr/0021-shared-instance-over-unix-socket.md) (shared
  instance), [ADR-0014](../../adr/0014-shutdown-flushes-pending-commits.md)
  (shutdown), [ADR-0016](../../adr/0016-tool-call-log-without-content.md)
  (tool-call log).
- Design D1, D12, D13, D21 in
  [`../../design/solution-design.md`](../../design/solution-design.md).
- SPEC-0003 (notes root), SPEC-0004 (`memctl mcp`), SPEC-0006 (instance lock),
  SPEC-0008 (git committer).
