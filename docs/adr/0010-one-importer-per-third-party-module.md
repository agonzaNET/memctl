# ADR-0010 - Confine each third-party module to one package, and enforce it with an import-scan test

## Status

Accepted

**Area:** Methodology

## Context

`memctl` is stdlib-first: the design allows two external modules, the official
Go MCP SDK and `gopkg.in/yaml.v3` ([ADR-0011](0011-mcp-sdk-and-yaml-v3.md)).
When a module is imported from many packages, its types leak into the domain
code, an upgrade touches the whole tree, and replacing it becomes a rewrite.
A rule written only in a guide is easy to break without noticing.

## Decision

> We will import each third-party module from exactly one package, and a test
> in `internal/arch` will fail on a second importer or on a module that no ADR
> allows.

- **Owners.** `github.com/modelcontextprotocol/go-sdk` → `internal/mcpserver`;
  `gopkg.in/yaml.v3` → `internal/parse`.
- **The test.** `internal/arch/imports_test.go` keeps the map of allowed
  modules and their owner package. It fails when non-test code in another
  package imports a module, when code imports a module not in the map, when
  `go.mod` requires a module not in the map directly, and when a module in the
  map is imported by nothing. A module enters the map in the change that first
  imports it. Tests may import a module from any package.
- **A new module** needs an ADR, a line in `go.mod` and a line in the map.
- **Pure Go.** The build is `CGO_ENABLED=0` ([ADR-0012](0012-build-targets-and-development-environment.md));
  a module that needs cgo is not allowed.

## Alternatives Considered

| Alternative | Reason not chosen |
|---|---|
| A rule in `docs/agents/backend.md` only | Nothing fails when it is broken. |
| A linter such as `depguard` | Another tool and config file for a check that a short stdlib test does. |
| Allow imports anywhere | Module types spread through the domain code. |

## Consequences

- **Positive:** Each module sits behind one package; upgrading or replacing it
  touches that package only.
- **Positive:** A new dependency cannot arrive without a visible change to the
  map.
- **Negative:** Code that needs a module's feature elsewhere goes through the
  owner package's own types.

## References

- [ADR-0002](0002-code-modularity.md), [ADR-0011](0011-mcp-sdk-and-yaml-v3.md),
  [ADR-0012](0012-build-targets-and-development-environment.md).
- `internal/arch/imports_test.go`.
