# Product Design — the source of truth

This directory holds the **authoritative product design** of `memctl`:
[`solution-design.md`](solution-design.md). It says what is built and why — the
goals, the note format, the architecture, the MCP tools, the decisions D1 to
D22 and the milestones. The specs, ADRs and code serve it.

## Verification records

Next to the design, keep **verification records**: one dated file per live
check that the design or a spec could not settle by a Go test — a first run
against a real notes root, a manual comparison of search results, a live check
of an edge case (e.g. `docs/design/YYYY-MM-DD-first-run-on-real-notes.md`). A record
says what was checked, what was found, and where each finding is settled. It is
written once and not updated afterwards: the settled behaviour lives in the
specs and the ADRs. List each record in this README.

None yet.

## Where it sits in the doc hierarchy

| Doc type | Answers | Authority |
|---|---|---|
| **Design** (here) | What is the product, and why, as a whole? | **Highest** — the source of truth |
| **Spec** (`docs/specs/`) | What must each capability do? Inputs, outputs, acceptance criteria. | Derives from the design |
| **ADR** (`docs/adr/`) | Why was each technical decision made? | Derives from the design |
| **Agent guide** (`docs/agents/`) | How do I work in this area of the code? | How-to, not what/why |

When `CLAUDE.md` (or any other doc) and the design disagree, **the design wins** —
update the other doc to match, or revise the design deliberately.

## Keeping it honest

The design is a living document, not a one-time artifact: when the product
direction changes, update the design **first**, then flow the change down into the
affected specs, ADRs, and code in the normal Spec → Plan → Implement → Verify
workflow ([`../agents/workflow.md`](../agents/workflow.md)).

Record amendments made after the design was written in an **Amendments**
section at the end of the design document, with the decision they change and
why — so a reader of a numbered decision never reads a statement the project
has since moved past without knowing it.

## Design decisions with their own ADR

A numbered design decision that a future reader is likely to question gets its
own ADR, with the reasoning and the alternatives.

| Design decision | ADR |
|---|---|
| Dependencies: go-sdk and yaml.v3 only (D1, D17) | [ADR-0011](../adr/0011-mcp-sdk-and-yaml-v3.md) |
| D18 — Build and dev environment | [ADR-0012](../adr/0012-build-targets-and-development-environment.md) |
| D19 — Tool text | [ADR-0013](../adr/0013-agent-first-tool-text.md) |
| D21 — Shutdown | [ADR-0014](../adr/0014-shutdown-flushes-pending-commits.md) |
| D22 — Tool results | [ADR-0015](../adr/0015-tool-result-shape.md) |
| Tool-call log ("Deployment and config") | [ADR-0016](../adr/0016-tool-call-log-without-content.md) |
