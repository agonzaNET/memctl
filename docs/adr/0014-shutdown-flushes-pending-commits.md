# ADR-0014 - Stop on stdin EOF or a signal with one shutdown context, and commit every queued path before exit

## Status

Accepted

> **Amended by [ADR-0021](0021-shared-instance-over-unix-socket.md)**: stdin
> EOF ends only the owner's own session; the shutdown starts when the last
> session of the shared instance has ended, or on a signal.

**Area:** Lifecycle

## Context

`memctl mcp` runs as long as its client keeps stdin open. Tool writes are
committed by a background goroutine after 2 s of quiet (D5). A client that
closes right after a write (the user quits Claude Desktop, a Claude Code
session ends) would end the process inside that window, and the write would
stay on disk but uncommitted, with no audit entry and no easy undo.

The process has two long-running parts: the MCP server reading stdin, and the
commit goroutine. Without one shared shutdown path, each would stop on its own
terms.

## Decision

> On stdin EOF, SIGINT or SIGTERM, `memctl mcp` cancels one shutdown context.
> The MCP server stops taking tool calls, the commit goroutine commits every
> queued path at once without waiting for the quiet window, and the process
> exits after that commit returns.

- `main` creates the context with `signal.NotifyContext` and passes it to the
  MCP server and the committer; stdin EOF cancels it too.
- A tool call in progress when shutdown starts finishes; no new call starts.
- A failed final commit is logged once at `ERROR` with the number of paths;
  the files stay on disk and show in `git status`.
- The instance lock is released by the kernel on exit (D12); there is no
  cleanup step.
- A second signal exits at once, without the final commit.

## Alternatives Considered

| Alternative | Reason not chosen |
|---|---|
| Commit synchronously after every write | One commit per write; loses the batching of D5 and slows bursts of edits. |
| Exit at once and leave queued paths | The last edits of a session would often miss git history. |
| Commit queued paths at the next start | The history would show edits at the wrong time, and edits made by hand in between would be mixed in. |

## Consequences

- **Positive:** Every tool write reaches git, even when the client closes at
  once.
- **Negative:** Shutdown can take as long as one `git commit`.

## References

- Design D5, D12, D21 in [`../design/solution-design.md`](../design/solution-design.md).
