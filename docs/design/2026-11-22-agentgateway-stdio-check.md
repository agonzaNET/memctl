# 2026-11-22 — agentgateway running `memctl mcp` as a stdio target

Design milestone 4 ("Remote use through agentgateway"). The question: can the
Raspberry Pi reach memctl through agentgateway's stdio MCP backend, with no
HTTP transport in memctl (ADR-0023, SPEC-0010, both deferred)?

## Setup

- Development VM, Linux arm64 (the Pi's architecture). No container, no
  OAuth: agentgateway listened on plain HTTP and a small Python client sent
  MCP streamable HTTP requests (`initialize`, `notifications/initialized`,
  `tools/list`, `tools/call`, `DELETE`) to `http://127.0.0.1:3000/mcp`.
- agentgateway v1.7.0, `agentgateway-linux-arm64`, sha256
  `85724a15dc9f5179fb6e2fd85b0ba2d4fea56dd289abb8bf8fefadc3bea8b89e`
  (matches the release's `.sha256`).
- memctl `0.3.2-c66d5dc`, built from `main` (the code of v0.3.2; later commits
  are docs only).
- A scratch notes root in a git repository with one note, `pa-850.md`.
- agentgateway config:

  ```yaml
  mcp:
    port: 3000
    targets:
    - name: memctl
      stdio:
        cmd: /tmp/agtest/memctl-wrap.sh   # exec memctl mcp --root <scratch> --log-level debug, stderr to a file per process
        args: []
  ```

## Checks

| # | Check | Result |
|---|---|---|
| 1 | `initialize` and `tools/list` through agentgateway | Pass. `serverInfo` `memctl 0.3.2-c66d5dc`; 8 tools, not prefixed (one target); `search` for `PA-850` found the note. |
| 2 | `note_write` through agentgateway, git history | Pass. Committed after the quiet window as `memory: note_write Scratch B.md` by `memctl <memctl@localhost>`. |
| 3 | Two client sessions at once | Pass. agentgateway starts **one child process per MCP session**. The first became the owner (`sharing` true); the second logged `forwarding` and the owner `session opened` `forward-1`. A note written in session B was read in session A. |
| 4a | Session B (the forwarder) ends with `DELETE` | Pass. The forwarder logged `stopped` at once (stdin EOF); the owner logged `session closed` and session A went on working. |
| 4b | SIGTERM to agentgateway 13 ms after a `note_write` | Pass. The owner made the final commit 22 ms later and logged `stopped`; no memctl process was left. |
| 4c | SIGKILL to agentgateway right after a `note_write` | Pass. The owner saw stdin EOF, committed and logged `stopped`; no memctl process was left. |
| 4d | Session A (the owner's own session) ends with `DELETE` while session B is open | **Fail, G1.** |
| 5 | Errors reported by agentgateway | Only the HTTP 500 of G1. |

## Findings

| ID | Finding | Settled |
|---|---|---|
| G1 | When the session whose child is the owner ends, agentgateway closes that child's stdin and ends the process 2–3 s later. The owner, which keeps running while forwarded sessions remain (ADR-0021), logs no `stopped` line: it gets no shutdown it can see (the signal could not be traced; ptrace is blocked in the VM). Its forwarders then log `memctl (pid N) serving the root stopped` and exit, and every other open session answers HTTP 500 until its client starts a new session. agentgateway starts sessions and ends them as clients come and go, so this breaks concurrent use whenever the first session ends. A tool write made in the last moments before the kill would stay on disk uncommitted until the next start, which commits it as an external change. | Open: a decision for the maintainer. |
| G2 | A session that a client abandons without `DELETE`: how long agentgateway keeps its child was not tested. | Open: to check on the Pi. |
| G3 | agentgateway generated a session key and wrote it to `config.session.key` next to the config, and recommends setting `SESSION_KEY`. | Observation for the deployment. |

## Follow-up: agentgateway settings

The agentgateway v1.7.0 config reference (`schema/config.md`) has two settings
that bear on G1 and G2:

- `config.mcp.sessionTtl`: "Time to live for MCP sessions before they are
  closed automatically. Defaults to 30 minutes." This answers G2: an abandoned
  session, and its child process, ends after the TTL. In stateful mode it is
  also a second cause of G1: an owner whose session reaches the TTL is ended
  while other sessions may still use it.
- `statefulMode` on the MCP backend (`stateful`, the default, or
  `stateless`): "Whether to keep a persistent session across requests
  (Stateful) or create one per request (Stateless)."

Checked with `statefulMode: stateless` (and `config.session.key: disabled`):

| # | Check | Result |
|---|---|---|
| 6 | `initialize`, `tools/list`, `note_write`, `note_read` | Pass. No `Mcp-Session-Id` header. One memctl process per request: each loaded the notes, served one call, made its final commit and logged `stopped`. The write was committed at once (`11d313f`); the tree stayed clean. |
| 7 | Six `search` calls sent in parallel | Pass. All six answered in 0.55 s in total; no errors; no process left. |

In stateless mode no memctl process outlives its request, so G1 cannot occur
unless a request runs longer than the 2–3 s agentgateway waits after a
session ends. The cost is a process start and a full load of the notes for
every request (milliseconds for 58 notes; it grows with the root), and a
commit per write instead of per batch.

## Verdict

agentgateway's stdio backend works with memctl for tools, writes, commits,
forwarding and shutdown. It is not yet usable for several concurrent
sessions in its default stateful mode, because of G1: memctl's owner is tied
to the first session's process, and agentgateway ends that process with the
session. With `statefulMode: stateless`, every check passed: G1 does not occur,
at the cost of one memctl start per request. Whether claude.ai works with a
server that returns no session ID is to be checked on the Pi.
