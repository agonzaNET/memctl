# adr.md — Architecture Decision Records

How ADRs are written and maintained here. An ADR captures **why** a significant
technical decision was made — the project's long-term memory. This guide holds the
**conventions**: when to write one, the status and supersession rules, the rules
for AI agents, and the controlled `Area` vocabulary. The mechanics (sections,
status values, format) live in [`TEMPLATE.md`](../adr/TEMPLATE.md); the
**registry** of which ADRs exist and their status lives in
[`../adr/README.md`](../adr/README.md) (ADR-0009), mirroring the spec
index. ADR-0001 records why we keep ADRs at all.

One file per decision: `docs/adr/<four-digit-index>-<short-slug>.md`. The index
is sequential and never reused, even when an ADR is superseded.

---

## Foundational ADRs — always in force

Some decisions are **global**: they apply to every non-trivial change regardless
of area. Honor them on any work without needing to be pointed at them. The
foundational set — ADR-0002 plus the methodology ADRs ADR-0003 through ADR-0007 —
is listed at the top of the
[ADR index](../adr/README.md#foundational-adrs--always-in-force).

Every other ADR is **area-local**. Find the ones relevant to your change by its
**Area** in the [ADR index](../adr/README.md#index). You do **not** need to read
all ADRs to make a change — read the foundational set above plus the ones for the
area you are touching (and, when implementing a spec, the specific ADRs that spec
names in its `Governing ADRs:` line).

---

## When to write an ADR

Write one when a decision:

- Affects the overall architecture or project structure.
- Involves a real trade-off between two or more alternatives.
- Would be surprising or hard to understand without context.
- Is likely to be questioned or revisited later.

Skip it for routine implementation details, trivially reversible choices, or
style already covered by a linter. **When in doubt, write one** — a short ADR
beats none.

An AI agent should **proactively draft an ADR** (and ask the developer to
confirm before treating it as `Accepted`) when it is about to recommend a new
library/framework/infrastructure component, change the project structure or
deployment model, or take an approach that conflicts with or extends an
existing ADR.

---

## Status & supersession

Status values and their meanings are in [`TEMPLATE.md`](../adr/TEMPLATE.md):
`Proposed → Accepted → Rejected | Deprecated | Superseded`.

**Never edit the body of an `Accepted` ADR to change the decision
retroactively.** Minor fixes (typos, broken links, clarifying wording) are
fine. When a decision actually changes:

1. Set the old ADR's status to `Superseded by ADR-NNNN` and add a short
   `Superseded by` pointer note above its `**Area:**` line.
2. Write a new ADR with the new decision, referencing the old one in its
   Context.
3. Add the new ADR to the index in [`../adr/README.md`](../adr/README.md), and
   move the superseded one from **Active** to **Superseded / Deprecated** there.

What a superseded ADR looks like: `Superseded by ADR-NNNN` on the `## Status`
line, a short pointer note above the `**Area:**` line that links the successor
and says in one sentence what changed, the body left untouched, and the row
moved into the **Superseded / Deprecated** table in
[`../adr/README.md`](../adr/README.md).

A decision that a *lighter* pointer covers — the successor extends or amends the
old one rather than replacing it — stays `Accepted` and gets an
`> **Amended by ADR-NNNN**` note instead, above its `**Area:**` line. Name the
amendment in the successor's title too (`… (amends ADR-NNNN)`).

---

## Rules for AI agents

- **Read existing ADRs before proposing architectural changes.** If one governs
  what you are doing, follow it — do not silently deviate. If you think it
  should be revisited, say so and propose a new ADR; do not ignore it. Use the
  [ADR index](../adr/README.md#index) to triage by **Area**, then read the bodies
  that apply.
- When creating one: copy [`TEMPLATE.md`](../adr/TEMPLATE.md) verbatim (do not
  omit sections), increment the highest existing index, set the **Area** from the
  controlled vocabulary below (add a new area there only if none fits), write
  **Context** before **Decision** to avoid post-hoc rationalisation, and list at
  least one real alternative in **Alternatives Considered** — if you cannot find
  one, ask the developer. Then add the row to
  [`../adr/README.md`](../adr/README.md).

---

## Area vocabulary

Every ADR carries an **Area** (declared as an `**Area:**` line in the ADR, under
its Status) drawn from a small controlled vocabulary, used for discovery
(ADR-0009). Current areas:

`Methodology`, `Build`, `Tools`, `Lifecycle`.

- `Methodology` — how the project is worked on: specs, ADRs, tests, commits,
  the changelog, versioning, dependency rules.
- `Build` — third-party modules, build targets, the toolchain and the
  development machine.
- `Tools` — the MCP tool surface: tool text, result shape, the tool-call log.
- `Lifecycle` — process start, the instance lock, shutdown.

Adding a new area (for example `Index`, `Format` or `Git` when those decisions
arrive) is a one-line edit to this list and to the list in
[`../adr/README.md`](../adr/README.md#maintaining-this-index) — it does not need
its own ADR. `scripts/check-docs.sh` fails if a row's Area in the
[ADR index](../adr/README.md#index) disagrees with the ADR file.
