# ADR-0009 - Keep one ADR index next to the ADRs, with an area per ADR and a list of foundational ADRs

## Status

Accepted

**Area:** Methodology

## Context

ADR-0001 records decisions as numbered files under `docs/adr/`. An agent or a
person who wants to change one part of the code needs to find the decisions
that govern that part without reading every ADR. A flat list of titles does
not say which decisions apply everywhere and which apply to one area.

A list of ADRs kept in a second place (for example in each agent guide) is a
second copy that drifts from the files. The ADR files themselves are the
source; the index only summarises them.

## Decision

> We will keep one ADR index, `docs/adr/README.md`, next to the ADRs. Every ADR
> declares an `**Area:**` from a controlled vocabulary, and the index lists the
> foundational ADRs that apply to every change at the top.

- **Area.** Each ADR has an `**Area:**` line under its Status. The vocabulary
  is kept in `docs/agents/adr.md`; adding an area is a one-line edit there and
  in the index, and needs no ADR.
- **Foundational ADRs.** ADR-0002 and the methodology set ADR-0003 to ADR-0007
  apply to every non-trivial change. The index names them at the top.
- **One index.** The table in `docs/adr/README.md` (ID, title, status, area) is
  the only list. Agent guides point to it and never copy it.
- **Checked.** `scripts/check-docs.sh` fails when a row's status or area
  disagrees with its ADR file, or when an ADR has no row. The same holds for
  the spec index in `docs/specs/README.md`.
- **Superseded ADRs** move to a second table in the index, so the active table
  stays the working set. Files and IDs are kept.

## Alternatives Considered

| Alternative | Reason not chosen |
|---|---|
| A flat list of titles only | Does not say which ADRs apply to a change; every reader scans all of them. |
| Per-area folders with their own numbering | Breaks ADR-0001's stable, never-reused IDs. |
| ADR lists inside each agent guide | A second hand-maintained copy that drifts from the files. |

## Consequences

- **Positive:** A reader finds the decisions for a change by area in one table.
- **Positive:** The index cannot drift silently; the docs check fails.
- **Negative:** Every new ADR, status change or area change touches the index
  in the same change.

## References

- [ADR-0001](0001-record-architecture-decisions.md) — numbered, never-reused IDs.
- [`../agents/adr.md`](../agents/adr.md) — conventions and the area vocabulary.
- `scripts/check-docs.sh` — checks 1, 2 and 4.
