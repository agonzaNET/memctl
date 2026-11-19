# ADR-0018 - Write the specs of a stage before any code, and plan a stage of coupled specs in one plan (amends ADR-0004)

## Status

Accepted

**Area:** Methodology

## Context

ADR-0004 pairs every spec with its own plan in the spec's folder, and lets
code start as soon as one spec and its plan are approved.

Two problems showed up before the first line of `memctl` code:

- **Coupled specs.** The milestone M1 specs (note parser, index and search,
  freshness check, MCP read tools, `memctl check`) only work together. One
  plan per spec builds one component at a time: the whole parser before the
  index exists, the whole index before any tool. What the later components
  need from the earlier ones is learned late, and the first end-to-end tool
  call comes last.
- **Contracts settled late.** Writing a spec while code for an earlier spec is
  already being built lets that code shape the later contract by accident,
  and a gap found in a later spec can force rework in finished code.

## Decision

> We will write and accept every spec of a stage before writing code for that
> stage, and plan a stage of coupled specs in one stage plan ordered as thin
> end-to-end slices. Specs stay one per capability. This amends ADR-0004.

- **Stage.** A set of specs that only deliver value together; for `memctl`, a
  design milestone (M1, M2, M3). A spec that stands alone is its own stage and
  keeps its own plan in its folder, as ADR-0004 says.
- **Specs first.** All specs of the planned release are written and accepted
  before implementation of any of its stages starts; for `memctl` v1, the
  specs of M1 and M2. M3 is build and release work governed by ADRs and needs
  no spec. The exception is a spec that cannot
  be written without code that does not exist yet (for example one that
  depends on measured behaviour or on an interface that the earlier code
  settles); it is written once that code is implemented, and its plan comes
  after it.
- **Stage plan.** `docs/plans/NNNN-slug/plan.md` (own sequential index, never
  reused), with the sections of ADR-0004: file map, dependency order, numbered
  tasks, definition of done. Every task names the requirements it implements
  as `SPEC-NNNN Rn`, so each requirement still maps to a task and a test.
  Tasks are ordered as thin slices that reach a working end-to-end result early.
- **Spec status.** A spec becomes `Implemented` when every one of its
  requirements has a passing test and its acceptance boxes are ticked, in
  whichever stage task that happens.
- **Replaced plans.** A per-spec plan that a stage plan takes over stays in its
  folder as history, with a note at the top that names the stage plan.

## Alternatives Considered

| Alternative | Reason not chosen |
|---|---|
| One plan per spec, code after each (ADR-0004 unchanged) | Builds coupled components one at a time; the first working tool call comes last. |
| One combined spec per stage | Loses the per-capability contract and its stable requirement IDs. |
| Specs and code interleaved, no batching | Earlier code shapes later contracts by accident. |

## Consequences

- **Positive:** Contracts of a stage are reviewed together before any code, so
  gaps between them are found on paper.
- **Positive:** The first end-to-end result arrives early in a stage.
- **Negative:** A stage plan is larger to review than one spec's plan, and a
  task can touch several specs.
- **Negative:** More time passes between the first spec and the first code.

## References

- [ADR-0004](0004-spec-plan-implement-verify-workflow.md), which this amends.
- [`../agents/workflow.md`](../agents/workflow.md), Phases 1 and 2.
