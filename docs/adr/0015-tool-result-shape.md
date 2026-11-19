# ADR-0015 - Return Markdown tool results with a one-sentence first line, and errors as a fixed code plus a next step

## Status

Accepted

**Area:** Tools

## Context

The callers of `memctl` are models. MCP already wraps every tool result in its
own structure: an `isError` flag and a list of content blocks. A JSON envelope
inside that (`status`, `message`, `data`) would add a second layer that a
model has to unpack, and `structuredContent` with an output schema per tool
costs schema upkeep for a reader that handles Markdown well.

Two properties of an envelope are still worth keeping: one short sentence that
says what happened, and a stable machine-readable error code that tests can
check and that the model can act on.

## Decision

> Tools return Markdown text, with no JSON envelope and no
> `structuredContent`. A success starts with one sentence that names what was
> done or found. An error has `isError` true and the text
> `<code>: <sentence that says what to do next>`, with the code from one fixed
> list.

- **Success:** `isError` is false. First line: one sentence, for example
  `Created "PAN-OS upgrade procedure" at runbooks/PAN-OS upgrade procedure.md.` Then the
  Markdown result (note, search hits, graph, listing).
- **Error:** `isError` is true. One text block, `<code>: <sentence>`, for
  example `conflict: The note changed since you read it; read it again and
  retry with the new sha.` The sentence names the next step: fix an argument,
  call another tool, or stop and tell the person.
- **Codes** are snake_case and live in one list in `internal/mcpserver`. A test
  fails when a tool emits a code outside the list. Each tool description says
  which codes it can return.
- **Protocol errors** (unknown tool, an input that breaks the schema) stay
  errors of the SDK.
- An error message never carries note content verbatim beyond what the agent
  needs to act (for example the current content on a `conflict`).

## Alternatives Considered

| Alternative | Reason not chosen |
|---|---|
| JSON envelope inside the text block | A second wrapper the model must parse; no gain over MCP's own structure. |
| `structuredContent` with an output schema per tool | Schema upkeep for every tool for a reader that handles Markdown well; can be added later without breaking text results. |
| Free-text errors with no code | Not testable, and the model cannot tell error kinds apart reliably. |

## Consequences

- **Positive:** One result shape for all tools; errors are testable by code.
- **Positive:** Results read well for the model with no parsing step.
- **Negative:** Programs that want structured data must parse Markdown until a
  later decision adds `structuredContent`.

## References

- Design D22 in [`../design/solution-design.md`](../design/solution-design.md).
- [`../agents/tools.md`](../agents/tools.md).
