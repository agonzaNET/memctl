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

## Verdict

agentgateway's stdio backend works with memctl for tools, writes, commits,
forwarding and shutdown. It is not yet usable for several concurrent
sessions, because of G1: memctl's owner is tied to the first session's
process, and agentgateway ends that process with the session.
