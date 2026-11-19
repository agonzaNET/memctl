# ADR-0019 - Take YAML support from go.yaml.in/yaml/v3, the maintained continuation of yaml.v3 (amends ADR-0011)

## Status

Accepted

**Area:** Build

## Context

ADR-0011 allows `gopkg.in/yaml.v3` for frontmatter. When the first code that
imports it was written, its repository was archived, and its last release,
v3.0.1, dates from 2022. The YAML organisation maintains the same package as
`go.yaml.in/yaml/v3`, with the same API and later releases (v3.0.6 at the time
of writing). A bug in the parser's only third-party dependency would get no
fix on the old path.

## Decision

> We will import YAML support from `go.yaml.in/yaml/v3` instead of
> `gopkg.in/yaml.v3`. Everything else in ADR-0011 stays: it is one of the two
> allowed modules, imported only by `internal/parse` (ADR-0010).

## Alternatives Considered

| Alternative | Reason not chosen |
|---|---|
| Keep `gopkg.in/yaml.v3` | Archived; no fixes. |
| Another YAML library | Different API and behaviour for no gain; the continuation is a drop-in. |

## Consequences

- **Positive:** Fixes and releases keep coming, with no code change beyond the
  import path.
- **Negative:** None known; the API is the same.

## References

- [ADR-0011](0011-mcp-sdk-and-yaml-v3.md), which this amends.
- [ADR-0010](0010-one-importer-per-third-party-module.md).
