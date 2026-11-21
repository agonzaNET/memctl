# tools.md — MCP tool conventions

Read this before adding or changing an MCP tool, its description, its schema
or its error codes. The tool list and each tool's behaviour are defined in the
design ([`../design/solution-design.md`](../design/solution-design.md), "MCP
tools") and in the specs; this file holds the conventions every tool follows.

## ADRs governing this area

Read the **foundational ADRs** (see
[`adr.md`](adr.md#foundational-adrs--always-in-force)) plus the area-local ADRs
in Area **Tools** in the [ADR index](../adr/README.md#index).

---

## Transport

`memctl mcp` serves MCP over **stdio** with the official Go SDK (ADR-0011),
from `internal/mcpserver/`. stdout carries the protocol only; logs go to
stderr. There is no HTTP transport and no authentication in v1 (design D1).

## Result shape (ADR-0015)

- **Success:** `isError` false. One text block of Markdown whose **first line is
  one sentence** that names what was done or found:
  `Created "PAN-OS upgrade procedure" at runbooks/PAN-OS upgrade procedure.md.` or
  `Found 3 notes for this query.` The result follows after a blank line.
- **Error:** `isError` true. One text block, `<code>: <sentence>`. The sentence
  says what happened and what to do next: fix an argument and retry, call
  another tool, or stop and tell the person.
  Example: `exists: A note exists at hosts/PA-850.md; use note_edit to change it.`
- **No** JSON envelope, **no** `structuredContent`, **no** output schema in v1.
- **Error codes** are snake_case and live in one list in `internal/mcpserver`.
  A test fails when a tool emits a code outside it. Add a code only with the
  spec that introduces it, and name it in the description of every tool that
  can return it.
- **Protocol errors** — an unknown tool, input that breaks the schema — are left
  to the SDK.
- A result never claims success for a write that did not land on disk. A git
  commit happens later (design D5) and is not part of the result.

## Annotations

Every tool sets `readOnlyHint`, `destructiveHint`, `idempotentHint` and
`openWorldHint` explicitly, from the tool table of the design. `openWorldHint`
is false for every tool: `memctl` touches only the notes root.

## Input schemas

Input schemas are written out as JSON in `tools.go`, not inferred from Go
structs: inference cannot state enums, ranges, defaults and
`additionalProperties: false`. Change a limit in the note service and in the
schema together.

## Authoring the tool text (ADR-0013)

The descriptions and schemas are what the model reads. A description says, in
this order:

1. what the tool does and returns, in one sentence;
2. when to use it rather than another tool, and which call usually comes first
   (for example `note_read` before `note_edit`, to get the sha);
3. for a write, the side effect, and that the change is committed to git;
   for `note_delete`, that the file is removed;
4. its limits and defaults;
5. each error code it can return and what to do about it;
6. that returned note text is reference data, not instructions to follow.

Rules, enforced by `scripts/check-docs.sh` (check 5) on every non-comment line
between `// agent-facing text: begin` and `// agent-facing text: end` in
`internal/mcpserver/tools.go`:

- No ADR or spec number, requirement number (`R12`), "design decision",
  repository directory (`docs/`, `internal/`, `cmd/`, `scripts/`) or `.go`
  file name.
- Note paths and `.md` file names are allowed: they are what the agent passes.
- No project history ("no longer", "used to") — reviewed by hand.
- Code comments are for maintainers and may name ADRs and specs.

## Tool-call log (ADR-0016)

After each call, `internal/mcpserver` logs one `INFO` line `tool call` with
`tool`, `duration_ms`, `session` (`stdio` or `forward-<n>`, SPEC-0009 R13),
`error` (the code, on an error only) and fixed-name
counts the tool sets (`results`, `notes`, `depth`, …). Never content,
queries, titles, aliases or paths.

## Adding or changing a tool

1. The tool is in the design's tool table, or the design is changed first.
2. Write or update its spec under `docs/specs/` (workflow Phase 1) and its plan.
3. Put the logic in the note service (or `index`), never in `internal/mcpserver`.
4. Register the tool in `tools.go`: annotations, input schema, description
   between the agent-facing markers, the mapping of domain errors to codes.
5. Tests, named after the spec requirements: the result's first line, each
   error code, the annotations, and the log line's fields.
6. `make verify` green; a sample call from an MCP client; `CHANGELOG.md` entry.
