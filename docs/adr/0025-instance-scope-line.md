# ADR-0025 - Tell the agent what each instance holds with a scope line in the server instructions and every tool description (amends ADR-0013, ADR-0024)

## Status

Proposed

**Area:** Tools

## Context

One `memctl` process serves one notes root (D4); several roots mean several
instances, each under its own MCP server name. The maintainer runs two side by
side in Claude Desktop: `infradocs` for work documents and `albdocs` for
personal ones.

Every instance sends the same fixed server instructions and the same eight tool
descriptions (ADR-0013). The only thing that tells an agent the instances apart
is the server name in the tool prefix (`infradocs__search`,
`albdocs__note_write`). A name such as `albdocs` says nothing about what
belongs there, so the agent guesses where to search and, worse, where to write
a note. A personal note written to the work root ends up in a work repository.

Two channels reach the agent from the server:

- **Server instructions**, sent once in the initialize result. Claude Desktop
  and Claude Code put them into the agent's system prompt. A Claude session
  linked to the desktop through the remote-devices bridge was observed on
  2026-11-23 not to receive them at all: its list of server instructions named
  other servers but neither memctl instance.
- **Tool descriptions**, which every client shows the agent with the tool,
  including that bridged session once it loaded the tools.

What belongs in an instance is known to the person who configures it, not to
the notes: the client configuration is the place to say it.

## Decision

We will add an optional, one-line **scope** to `memctl mcp`, set with
`--scope TEXT` or `MEMCTL_SCOPE`, and append it, after the fixed text, as one
line to the server instructions **and** to every tool description:

```text
Scope of this memory, set by the person who runs it: Alberto's personal notes: family, homelab, hobbies. No appliedAI work.
```

The fixed part of that line is agent-facing text and sits between the markers
in `tools.go` (ADR-0013); the scope itself is configuration, so it is checked
by the flag rules (one line, at most 200 characters, no control characters),
not by `scripts/check-docs.sh`. Without a scope, the instructions and
descriptions stay byte for byte as they are.

A shared instance (ADR-0021) serves every forwarded session with the owner's
server, so the owner's scope applies to all of them. The handshake carries each
side's scope, and a difference is logged as a `WARN` on both sides, like a
version difference, without refusing the session: a scope is guidance for the
agent, not a property of the data such as the git author.

The Claude Desktop extension (ADR-0024) gains an optional `scope` field,
passed to the server as `MEMCTL_SCOPE`; an empty value means no scope.

## Alternatives Considered

| Alternative | Reason not chosen |
|---|---|
| Scope in the server instructions only | Cheapest, but it does not reach an agent whose client drops the instructions, as the bridged session did. |
| A scope note inside the notes root (for example a `README.md`) | The agent sees it only if it searches for it, and note text is reference data, not instructions (ADR-0013); anyone who can write a note could change the instance's scope. |
| A standing instruction in each client (Claude preferences, `CLAUDE.md`) | Works without code, but has to be repeated and kept in sync per client, and travels with the client rather than with the server. |
| A more telling server name only | Free and still worth doing, but a name cannot say what does not belong there. |
| A separate `about` tool that returns the scope | The agent has to think of calling it before choosing an instance; a description is read without a call. |
| The scope in every tool result | Arrives after the agent has already chosen an instance, and costs tokens on every call. |
| A config file holding the scope | v1 has no config file (design, "Deployment and config"); one flag does not justify one. |

## Consequences

- **Positive:** The agent reads what each instance holds wherever it sees the
  tools, before it chooses one; the client configuration, not the notes, says it.
- **Positive:** No change for an instance without a scope: same text, same
  results, same tests.
- **Negative:** The scope is repeated nine times (instructions and eight
  descriptions); at the 200-character limit that is roughly 600 tokens per
  instance, at the 90-character example about 300.
- **Negative:** It guides the agent; it enforces nothing. A note can still be
  written to the wrong instance and has to be moved by hand.
- **Risk / Note:** The descriptions are no longer fully static: check 5 of
  `scripts/check-docs.sh` sees the fixed text only, and the scope is whatever
  the person configures, so its wording is theirs.
- **Note:** `memctl serve` (ADR-0023, SPEC-0010, deferred) takes the same flag
  when it is built.

## References

- Design D4 (one notes root per instance), D19 (tool text).
- ADR-0013 (agent-first tool text), ADR-0021 (shared instance), ADR-0024
  (Claude Desktop extension).
- SPEC-0011 (instance scope).
