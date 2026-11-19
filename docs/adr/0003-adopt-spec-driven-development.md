# ADR-0003 - Adopt Spec-Driven Development with specs under docs/specs

## Status

Accepted

**Area:** Methodology

## Context

This project documents the *why* of decisions in ADRs (`docs/adr/`) and the *how*
of working in each area in agent guides (`docs/agents/`). A third artifact is
needed to capture the *what* — the observable behaviour each capability must
exhibit: its scope, inputs, outputs, and acceptance criteria.

Without that, behaviour is only specified implicitly by the code itself. New
work (human or AI agent) has to reverse-engineer intent from the implementation,
and there is no artifact to check a change against other than "does it still
build / look right". That gap widens as the codebase grows, so the practice is
adopted from the start.

## Decision

> We will adopt Spec-Driven Development (SDD): each capability has a
> specification under `docs/specs/`, and code is expected to satisfy its spec.

- Specs live in `docs/specs/`, **one folder per capability**, numbered
  sequentially (`<four-digit-index>-<slug>/`) — mirroring the ADR convention.
  The spec is `spec.md` inside that folder; its plan and any supporting files
  for the capability live there too.
- Every spec uses `docs/specs/SPEC-TEMPLATE.md` and states **scope**,
  **inputs**, **outputs**, numbered testable **requirements**, **error
  handling**, and **acceptance criteria** ("done looks like").
- Specs describe *what*, not *why*. Each links to the governing ADR(s); it does
  not restate rationale.
- When a capability with a spec changes, the spec is updated in the same change.
  When behaviour and spec disagree, one of them is treated as a bug.
- Specs are distinct from ADRs (decisions/rationale) and agent guides
  (how to work in the code). All three coexist.

## Alternatives Considered

| Alternative | Reason not chosen |
|---|---|
| Keep behaviour implicit in code + ADRs only | No artifact to validate a change against; intent must be reverse-engineered; acceptance criteria live only in reviewers' heads. |
| Extend ADRs to also carry behavioural specs | Conflates *why* (decision, immutable once Accepted) with *what* (behaviour, evolves with the code); ADRs are explicitly not meant to be edited to track behaviour. |
| Encode behaviour only as automated tests | Tests assert behaviour but read poorly as a human-facing contract. Specs and tests are complementary — specs can later seed tests. |
| Put specs in agent guides (`docs/agents/`) | Agent guides are *how to work here* references, not per-capability contracts; mixing the two dilutes both. |

## Consequences

- **Positive:** Each capability has an explicit, testable contract with
  acceptance criteria, independent of the implementation.
- **Positive:** New work has a target to build to and to validate against;
  reviews can check code against the spec.
- **Positive:** Specs can later seed automated tests.
- **Negative:** Specs are another artifact to keep in sync; a stale spec is
  worse than none. Mitigated by requiring spec updates in the same change as the
  behaviour change.
- **Risk / Note:** SDD's value depends on discipline. If specs drift from code
  they become misleading. The "update the spec in the same change" rule and the
  CLAUDE.md pointer are the guardrails.

## References

- `docs/specs/README.md` — spec index and conventions
- `docs/specs/SPEC-TEMPLATE.md` — the template all specs use
- ADR-0001 — Record architecture decisions (the analogous decision for ADRs)
