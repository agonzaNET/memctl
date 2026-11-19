# ADR-0004 - Adopt the Spec → Plan → Implement → Verify workflow

## Status

Accepted

<!--
Status values: Proposed | Accepted | Rejected | Deprecated | Superseded by ADR-NNNN
-->

**Area:** Methodology

## Context

ADR-0003 adopted Spec-Driven Development: every capability has a spec under
`docs/specs/` describing *what* it must do. But nothing defines the process that
connects an approved spec to shipped code. Work could jump from a spec — or from
no spec at all — straight to ad-hoc implementation, with planning and
verification living only in the author's head.

As the codebase grows, and as AI agents do more of the implementation, an
explicit, gated workflow keeps changes reviewable, prevents scope drift, and
gives a human clear checkpoints to intervene before code is written rather than
after.

The workflow must not become ceremony for trivial edits, so it needs an explicit
trigger for when the full process applies.

## Decision

> Non-trivial changes follow a four-phase cycle: **Spec → Plan → Implement →
> Verify.** The operational detail lives in `docs/agents/workflow.md`; this ADR
> records the decision and its boundaries.

- **Spec (what)** — `docs/specs/NNNN-slug/spec.md`, per ADR-0003 and `SPEC-TEMPLATE.md`.
- **Plan (how)** — `docs/specs/NNNN-slug/plan.md`, alongside its spec in the same
  capability folder:
  a file map, a dependency-ordered numbered task list whose tasks name the spec
  requirements (`R1`, `R2`, …) they implement, and the build/test command that
  defines done. Plan files are **committed and kept** as an audit trail, not
  deleted after implementation.
- **Implement** — task by task, one task per commit.
- **Verify** — run tests and linter and show the output as evidence before
  declaring done; confirm every spec requirement has a passing test that names it
  and tick the spec's acceptance criteria against that evidence; open a PR only
  when all tasks are checked off and the build is green.
- **Trigger.** The full workflow applies when a change touches more than two
  files, cannot be described in one sentence, touches a security boundary /
  configuration / a public interface or contract / a data shape, or is expected
  to take more than ~30 minutes. Trivial changes (typos, renames, a single log
  line, an obvious one-file diff) skip it.
- **Review gates.** The Spec and Plan phases are drafted in plan mode and stop for
  human review before any code is written. An agent that finds the plan is wrong
  mid-implementation stops and surfaces the conflict rather than silently
  deviating.

Test-first development (Phase 3) and commit conventions (Phase 4) are decided
separately in ADR-0005 and ADR-0006.

## Alternatives Considered

| Alternative | Reason not chosen |
|---|---|
| Spec only, then ad-hoc implementation | Leaves the spec-to-code gap unmanaged: no planning artifact, no verification discipline, no pre-code review gate. |
| Plans as ephemeral plan-mode output, not committed | Loses the audit trail and the shared handle that multi-agent / multi-session work needs to coordinate on. |
| Apply the full workflow to every change | Too much ceremony for a small repo; a one-line fix should not require a spec and a plan. Hence the explicit trigger. |
| Track tasks in an external tool (issues/board) | Adds a system outside the repo; keeping the plan beside the spec keeps the contract and the task list versioned together. |

## Consequences

- **Positive:** Every non-trivial change has an explicit path from contract to
  shipped code, with human checkpoints before implementation.
- **Positive:** Plan files give reviewers and other agents a single source of
  truth for what is being built and how far along it is.
- **Negative:** More artifacts to create and keep in sync (a plan per non-trivial
  change). Mitigated by the trigger criteria and by pairing plans with specs.
- **Risk / Note:** A stale plan is misleading. Plans are working documents — once
  every task is checked off and verified, the plan is a historical record, not a
  live contract (the spec remains the live contract).

## References

- ADR-0003 — Adopt Spec-Driven Development (the *what*; this ADR adds the process)
- ADR-0005 — Test-first development (Phase 3 detail)
- ADR-0006 — Conventional commit messages referencing the spec (Phase 4 detail)
- `docs/agents/workflow.md` — the operational guide
