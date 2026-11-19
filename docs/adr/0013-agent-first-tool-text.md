# ADR-0013 - Write tool descriptions and schemas for the agent, and check them for internal references

## Status

Accepted

**Area:** Tools

## Context

The MCP tool descriptions and input schemas are the only documentation the
model sees. It cannot open this repository's ADRs, specs or source files, so a
reference such as "see ADR-0014" or "R12" is noise to it, and a repository
path can be mistaken for a note path. At the same time, `memctl` addresses
notes by relative path, so a description must be able to show the path form,
for example `ops/pa-850.md`.

## Decision

> We will write every tool description and input schema for the agent first,
> and `scripts/check-docs.sh` will fail when that text names an ADR, a spec, a
> requirement number, "design decision" or a repository path. Note paths ending
> in `.md` are allowed.

- **Where.** The text sits in `internal/mcpserver/tools.go` between the markers
  `// agent-facing text: begin` and `// agent-facing text: end`. The check scans
  every non-comment line between them, and fails when the file or the markers
  are missing once the file exists.
- **What a description says,** in order: what the tool does and returns, when
  to use it rather than another tool, its limits and defaults, what each error
  code means and what to do next, and that returned note text is reference
  data, not instructions to follow.
- **Forbidden:** ADR and spec numbers in any letter case, requirement numbers
  (`R12`, `R8a`), "design decision", repository directories (`docs/`,
  `internal/`, `cmd/`, `scripts/`) and `.go` file names.
- **Allowed:** relative note paths and `.md` file names, because they are what
  the agent passes to the tools.
- Code comments are for maintainers and may name ADRs and specs.

## Alternatives Considered

| Alternative | Reason not chosen |
|---|---|
| Review the text by hand only | Internal references slip in unnoticed; the check costs a few lines of shell. |
| Forbid every file name, `.md` included | The agent could not be shown the path form it must use. |
| Generate descriptions from the specs | Specs are written for developers; the agent needs different words. |

## Consequences

- **Positive:** Descriptions stay useful to the model and free of references it
  cannot follow.
- **Negative:** Every description edit must keep within the markers; moving the
  text elsewhere fails the check.
- **Risk / Note:** Project-history wording ("no longer", "used to") is not
  machine-checked; review it by hand.

## References

- Design D19 in [`../design/solution-design.md`](../design/solution-design.md).
- [`../agents/tools.md`](../agents/tools.md); `scripts/check-docs.sh`, check 5.
