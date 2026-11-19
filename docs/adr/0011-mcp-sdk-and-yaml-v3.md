# ADR-0011 - Use the official Go MCP SDK and yaml.v3 as the only third-party modules

## Status

Accepted

**Area:** Build

## Context

`memctl` is one static Go binary with no runtime dependencies (design goals,
D1, D17). Two parts would be costly and risky to write by hand:

- **MCP over stdio.** The protocol has a JSON-RPC framing, an initialisation
  handshake, capability negotiation, tool schemas and annotations, and it
  still changes between protocol versions.
- **YAML frontmatter.** Note frontmatter is YAML. A hand-written subset parser
  would reject or misread valid frontmatter that editors and other tools
  write, and unknown keys must be preserved untouched.

Everything else (Markdown structure, BM25F, the link graph, git) is done with
the standard library or the `git` CLI (D17, D5).

## Decision

> We will use `github.com/modelcontextprotocol/go-sdk` for the MCP server over
> stdio, and `gopkg.in/yaml.v3` for frontmatter. No other third-party module
> is used in v1.

- The MCP SDK is imported only by `internal/mcpserver`, yaml.v3 only by
  `internal/parse` ([ADR-0010](0010-one-importer-per-third-party-module.md)).
- Both are pure Go, so the build stays `CGO_ENABLED=0`.
- Versions are pinned in `go.mod`; an upgrade is its own change with the full
  test suite.

## Alternatives Considered

| Alternative | Reason not chosen |
|---|---|
| Hand-written MCP framing | Re-implements a moving protocol; every protocol change becomes our bug. |
| A community MCP library | The official SDK tracks the specification and is maintained with it. |
| A hand-written YAML subset | Misreads valid frontmatter and cannot round-trip unknown keys safely. |
| TOML or JSON frontmatter | Breaks compatibility with existing notes and Obsidian (D2, lenient reads). |

## Consequences

- **Positive:** Protocol and YAML edge cases are handled by maintained code.
- **Negative:** Two external release cycles to follow; an SDK change can
  require code changes in `internal/mcpserver`.

## References

- Design D1, D2, D17 in [`../design/solution-design.md`](../design/solution-design.md).
- [ADR-0010](0010-one-importer-per-third-party-module.md).
