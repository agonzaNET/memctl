# ADR-0021 - Share one memctl process per notes root: the first process owns it, later ones forward over a Unix socket (amends ADR-0014)

## Status

Proposed

**Area:** Lifecycle

## Context

Design D12 allows exactly one `memctl mcp` process per notes root, so that
there is one writer, one index and one git committer. A second process on the
same root exits with `root <path> is in use by memctl (pid N)`. The design
accepted this for v1 ("one client per root") and listed sharing a root as an
HTTP backlog item after v2.

Claude Desktop now starts each server of `claude_desktop_config.json` twice
from one entry: one copy for chats and one in a separate pool for Cowork and
Code sessions. Its log of 2026-11-20 and 2026-11-21 shows the pool copy
refused by the lock and retried about every 60 s
(`Couldn't start for Cowork and Code sessions`, context `shared-pool`). The
notes work in chat and are missing in Cowork, where the maintainer's real-use
trial runs.

Constraints:

- Clients start `memctl` as a stdio child process (D1). Desktop's local config
  starts commands only; a server added by URL is reached from Anthropic's
  cloud and cannot reach `localhost`.
- The MCP stdio shutdown sequence lets a client close stdin, then send
  SIGTERM, then SIGKILL; a process cannot count on outliving its client.
- The note service already serialises every call behind one mutex, and the Go
  MCP SDK serves several sessions on one server.

## Decision

> We will keep one process per root (the owner) and let every later
> `memctl mcp` on that root forward its stdio to the owner over a Unix socket,
> because it fixes multi-client use for any stdio client with no client
> configuration, keeps D12's single writer and committer, and needs no
> authentication beyond file permissions.

- **Owner.** The process that takes the instance lock is the owner. After the
  startup load it listens on `<root>/.memory/memctl.sock` (mode 0600, in the
  0700 `.memory/` folder) and serves its own stdio and every accepted
  connection as separate MCP sessions of one server, over one note service and
  one git committer.
- **Forwarder.** A process that finds the lock held loads nothing. It
  connects to the socket, sends a one-line handshake, and then copies bytes
  between its stdio and the connection. It retries for up to 10 s (and takes
  over as owner when the lock comes free in that time), then exits with the
  lock message.
- **Handshake.** One JSON line each way before any MCP byte, with a protocol
  number. The owner refuses a forwarder whose `--git-author` differs from its
  own, because every commit carries the owner's author (D10, D13). A
  different version only logs a `WARN`: a forwarder interprets no MCP bytes.
- **Lifetime (amends ADR-0014).** Stdin EOF ends only the owner's own session.
  The owner keeps serving while any session is open, and starts the shutdown
  of ADR-0014 when the last session has ended, or at once on SIGINT or
  SIGTERM. A forwarder exits when its stdin ends or when the owner goes away;
  its client restarts it, and the restarted process becomes owner or
  forwarder.
- **No sharing when the socket cannot be created** (a socket path longer than
  the platform allows, about 103 bytes on macOS, or any bind error): the owner
  logs a `WARN` and works as before this decision.
- `memctl check` is unchanged. No flag, environment variable or third-party
  module is added.

## Alternatives Considered

| Alternative | Reason not chosen |
|---|---|
| Keep refusing the second process (D12 as written) | Notes are missing in every client surface after the first; other tools that shipped a singleton lock removed it for this reason. |
| Many processes on one root, a lock per write instead of per process (Basic Memory's model) | Each process would see the other's writes as external changes; who commits a change, and under which author, needs a new design, and indexes and freshness walks are duplicated. |
| Detached background daemon, every client process a forwarder (gopls `-remote=auto`) | Needs process spawning, an idle timer, a separate log file, and handling for an old daemon that outlives an upgrade; a process outside any client's control. |
| Streamable HTTP with a generated token (design backlog) | Desktop needs a stdio bridge in front of it anyway; nothing starts the server; loopback TCP needs a token store, rotation and `Origin` checks. Kept for remote and team use; it can later attach to the owner. |
| Socket in a per-user temporary folder under a hashed name | No length limit, but the socket lives outside the root and is harder to find; the root's `.memory/` already holds the lock. |
| Forwarder takes over in place when the owner dies, replaying the client's `initialize` | Seamless failover, but the forwarder must parse JSON-RPC and a call in flight is lost anyway; restart by the client is enough. |
| Refuse on any setting difference, or on none | Any: upgrading the binary while a client runs breaks the other clients until a restart. None: commits can land under an author that was never configured for that client. |

## Consequences

- **Positive:** Claude Desktop chat and Cowork, or any set of stdio clients on
  one machine, share one root with no client configuration change.
- **Positive:** D12's guarantees hold: one writer, one index, one git committer
  per root.
- **Positive:** Authentication is the file system's: only the owner's user can
  reach `.memory/`.
- **Negative:** When the owner is killed, its forwarded sessions end until their
  clients restart them (Claude Desktop retries about every 60 s).
- **Negative:** The process lifetime is no longer tied to one client: an owner
  whose own client closed stdin keeps running while forwarded sessions remain,
  unless the client sends SIGTERM.
- **Negative:** One more file in `.memory/`, and a root whose socket path is too
  long cannot be shared.
- **Note:** The `tool call` log line gains a `session` field (no content), so the
  owner's log tells the sessions apart (ADR-0016 is unchanged in spirit).

## References

- Design D1, D10, D12, D13, D21 in [`../design/solution-design.md`](../design/solution-design.md).
- [ADR-0014](0014-shutdown-flushes-pending-commits.md) (shutdown), amended here.
- SPEC-0009 (shared instance), SPEC-0006 (instance lock).
- Gopls daemon mode, the forwarder pattern this follows:
  <https://tip.golang.org/gopls/daemon>.
- MCP transports, stdio shutdown sequence:
  <https://modelcontextprotocol.io/specification/2025-06-18/basic/transports>.
