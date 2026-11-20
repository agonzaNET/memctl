# SPEC-0006 - Instance lock

## Status

Accepted

> Amended on 2026-11-20 for finding L15 of the live check
> ([`2026-11-20-live-check.md`](../../design/2026-11-20-live-check.md)),
> delivered by stage plan 0004: R5 changed. Until the stage is implemented,
> the code matches the version of 2026-11-19.

<!--
Status values: Draft | Accepted | Implemented | Superseded by SPEC-NNNN | Deprecated
Lifecycle: Draft (under review) → Accepted (approved; the code may be built,
but at least one acceptance criterion, for example a live test, is still
open) → Implemented (shipped; code satisfies this spec and all acceptance
criteria are ticked). Superseded by SPEC-NNNN and Deprecated are terminal.
-->

**Governing ADRs:** ADR-0002, ADR-0005, ADR-0012, ADR-0014

## Purpose

Make sure that at most one `memctl mcp` process serves a notes root at a time,
so there is one writer and one git committer per root (design D12). A second
process on the same root stops at once and says which process holds it.

## Scope

**In scope**

- `internal/note`: taking, holding and releasing the lock file
  `<root>/.memory/memctl.lock`, and creating `<root>/.memory/` when it is
  missing.
- The start-up order of `memctl mcp` around the lock.

**Out of scope**

- Sharing one root between several clients (design backlog: HTTP mode).
- Network file systems on which `flock` does not work, and Windows (ADR-0012).
- `memctl check`, which takes no lock (SPEC-0005 R1).

## Inputs

- The resolved notes root (SPEC-0003 R1).

## Outputs

- The lock file, holding the PID of the process that holds the lock, followed
  by a newline.
- `<root>/.memory/.gitignore` with the single line `*`, so nothing in
  `.memory/` is ever committed.
- On refusal: exit status 1 and one `ERROR` log line with the message
  `root <path> is in use by memctl (pid <N>)`; the message is also printed as a
  plain line when stderr is a terminal.

## Behaviour & Requirements

- **R1** — `memctl mcp` MUST take the lock after resolving the root and before
  the startup load (SPEC-0003 R8), and MUST hold it until the process exits.
- **R2** — When `<root>/.memory/` does not exist, it MUST be created with mode
  0700, and `.memory/.gitignore` with the content `*` and a newline, mode 0600.
  An existing `.memory/.gitignore` MUST NOT be changed.
- **R3** — The lock MUST be taken with a non-blocking exclusive `flock` on
  `.memory/memctl.lock`, opened with `O_CREATE`, mode 0600, and close-on-exec.
- **R4** — After taking the lock, the process MUST truncate the file and write
  its PID and a newline.
- **R5** — When the lock is held by another process, `memctl mcp` MUST read the
  PID from the file, log one `ERROR` line with the message
  `root <path> is in use by memctl (pid <N>)` (or
  `root <path> is in use by another memctl process` when the PID cannot be
  read), and exit with status 1 without loading the notes. `<path>` MUST be the
  root as given; when the resolved root differs, it MUST follow in parentheses
  (`root /tmp/notes (/private/tmp/notes) is in use …`). The message MUST also
  be printed as a plain line to stderr only when stderr is a terminal
  (character device), so a client's log holds it once.
- **R6** — When `.memory/` or the lock file cannot be created or opened (for
  example a read-only root), `memctl mcp` MUST exit with status 1 and an
  `ERROR` line that names the path and the operation.
- **R7** — The lock file MUST NOT be deleted, on exit or at any time; the kernel
  releases the lock when the process ends, also after a crash or `SIGKILL`.
- **R8** — A process MUST be able to take the lock at once after the previous
  holder ended in any way, without a stale-lock cleanup step.

## Error handling

| Failure | Result |
|---|---|
| Lock held by another process | Message with the holder's PID, status 1 (R5) |
| `.memory/` or the lock file cannot be created or opened | `ERROR` naming the path, status 1 (R6) |
| The PID cannot be written after the lock is taken | One `WARN` line; the process goes on, holding the lock |

## Acceptance criteria — "Done looks like"

- [x] **R1** — An end-to-end test: the lock is held before the `started` line,
  and a second process started during the load of the first is refused.
- [x] **R2** — A root without `.memory/` gets the folder, its mode, and
  `.memory/.gitignore`; an existing `.memory/.gitignore` with other content is
  unchanged; `git status` in the root shows nothing from `.memory/`.
- [x] **R3, R4** — Two processes on one root: the second exits with status 1;
  the lock file holds the first one's PID.
- [x] **R5** — The second process's stderr, captured through a pipe, holds the
  message once, in the `ERROR` line, with the first one's PID; with an empty
  lock file, the message without a PID; a root given through a symlink shows
  the given path and the resolved one in parentheses; a root given as resolved
  shows it once.
- [x] **R6** — A read-only root gives status 1 and the `ERROR` line.
- [x] **R7, R8** — After the holder is killed with `SIGKILL`, a new process takes
  the lock at once; the lock file still exists after every kind of exit.

## References

- Design: D12, "Change detection, concurrency and git" in
  [`../../design/solution-design.md`](../../design/solution-design.md).
- SPEC-0003 (notes root), SPEC-0004 (`memctl mcp`), SPEC-0005 (`memctl check`).
- Live check of 2026-11-20, finding L15:
  [`2026-11-20-live-check.md`](../../design/2026-11-20-live-check.md).
