# Specifications

This directory holds the **specifications** (specs) for this project, following a
Spec-Driven Development (SDD) workflow: the spec describes *what* a capability
must do — its scope, inputs, outputs, and what "done" looks like — independently
of *how* it is implemented. Code is expected to satisfy its spec; when behaviour
and spec disagree, one of them is a bug.

## How specs relate to the other docs

| Doc type | Answers | Lives in |
|---|---|---|
| **Design** | What is the product, and why, as a whole? | `docs/design/` |
| **Spec** (here) | *What* must the system do? What are the inputs, outputs, and acceptance criteria? | `docs/specs/` |
| **ADR** | *Why* was a technical decision made? What were the alternatives? | `docs/adr/` |
| **Agent guide** | *How* do I work in this area of the code? | `docs/agents/` |

Each spec links to the ADR(s) that justify its decisions. Specs do not restate
the rationale — they reference the ADR and focus on observable behaviour.

## Conventions

- One **folder** per capability: `docs/specs/<four-digit-index>-<short-slug>/`.
  Every file generated for a spec lives in that folder — the spec itself
  (`spec.md`), its implementation plan (`plan.md`), and any supporting material
  (diagrams, research notes, data samples). The folder keeps a capability's
  artifacts together and lets the plan and spec stay paired by location rather
  than by filename suffix.
- The index is sequential and never reused (the folder's four-digit prefix is
  the spec ID).
- Every spec's `spec.md` uses [`SPEC-TEMPLATE.md`](SPEC-TEMPLATE.md).
- Requirements are written so they can be turned into a test. Each is numbered
  (`R1`, `R2`, …) and uses **MUST / MUST NOT / SHOULD** deliberately. The IDs are
  stable handles: plan tasks, tests, and acceptance criteria all refer to a
  requirement by its ID, so each requirement can be traced to the test that
  proves it (see [`../agents/workflow.md`](../agents/workflow.md)).
- Acceptance criteria ("Done looks like") are concrete and observable — a
  reviewer can check each one against a running `memctl` or the code — and every
  requirement has at least one acceptance criterion citing its ID.

## Status & lifecycle

A spec's `Status` field moves through a defined lifecycle (the vocabulary lives
in [`SPEC-TEMPLATE.md`](SPEC-TEMPLATE.md)):

- **Draft** → under review; not yet approved. Phase 1 of the workflow.
- **Accepted** → approved; the contract is settled. The code may be built, but
  at least one acceptance criterion (for example a live test) is still open.
- **Implemented** → shipped; code satisfies the spec and every acceptance
  criterion is ticked. Set in Phase 4 (see
  [`../agents/workflow.md`](../agents/workflow.md)).
- **Superseded by SPEC-NNNN** and **Deprecated** are terminal (see below).

When behaviour and spec disagree, one of them is a bug — fix the code or correct
the spec in the same change; never let them drift.

### Retiring or superseding a spec

A spec is a permanent record of a capability's contract. Once it is `Accepted`
or `Implemented`, do **not** rewrite its body to erase a capability or to match
behaviour you have dropped. Retire it explicitly instead, mirroring the ADR rule
in [`../agents/adr.md`](../agents/adr.md):

1. **Replaced by a new capability** — write the new spec in its own
   `NNNN-slug/` folder (next sequential index, never reused). Set the old spec's
   status to `Superseded by SPEC-NNNN`, add a one-line note at the top of the old
   spec linking the successor, and have the new spec reference the old one in its
   Scope/References. Keep the old folder as history.
2. **Removed entirely (not replaced)** — set the spec's status to `Deprecated`,
   add a note stating why and when the capability was removed, and record a
   `Removed` entry in `CHANGELOG.md`. Keep the spec file as a historical record.
3. **Update the index row** to the new status in the same change. Minor fixes to
   a retired spec (typos, broken links) are fine; changing what it said is not.

`scripts/check-docs.sh` fails if a spec file's status and its index row ever
disagree, so these steps are enforced, not just recommended.

## Index

One row per capability spec. Keep each row's status in sync with the `Status`
field in the spec file itself (the workflow sets a spec to `Implemented` in
Phase 4 — see [`../agents/workflow.md`](../agents/workflow.md)), enforced by
`scripts/check-docs.sh`. The design's milestones (M1 read-only memory, M2
writes, M3 release) are delivered as one or more specs each.

| ID | Title | Status | Governing ADRs |
|---|---|---|---|
| SPEC-0001 | Note parser | Accepted | ADR-0002, ADR-0005, ADR-0010, ADR-0011 |
| SPEC-0002 | In-memory index: link resolution, search, graph walk and listing | Accepted | ADR-0002, ADR-0005, ADR-0017 |
| SPEC-0003 | Notes root, startup load and freshness check | Accepted | ADR-0002, ADR-0005, ADR-0016, ADR-0017 |
