# ADR-0006 - Conventional commit messages referencing the spec

## Status

Accepted

<!--
Status values: Proposed | Accepted | Rejected | Deprecated | Superseded by ADR-NNNN
-->

**Area:** Methodology

## Context

Free-form commit history loses the link between a change and the reason for it.
With SDD (ADR-0003) and the Spec → Plan → Implement → Verify workflow (ADR-0004),
each commit corresponds to one numbered plan task that implements part of a spec.
Linking the commit back to its spec turns the history into a navigable trail from
contract → task → change, which is most of the point of adopting SDD in the first
place.

If the repo or harness mandates a commit trailer (for example a `Co-Authored-By:`
line), this convention must coexist with it rather than replace it.

## Decision

> Commit subjects follow Conventional Commits and reference the governing spec:
>
> ```
> <type>(<scope>): <short description> (spec: SPEC-NNNN)
>
> <body, if needed>
>
> <required trailer(s), if any>
> ```

- Types: `feat | fix | refactor | test | chore | docs`.
- `(spec: SPEC-NNNN)` cites the governing spec; omit it only for changes with no
  spec (trivial edits, docs-only changes).
- Commit after **each completed plan task**, not once per feature.
- The failing-test commit from the TDD loop (ADR-0005) is marked as such, e.g.
  `test(parse): failing test for fenced headings (spec: SPEC-0001)`, so a red
  commit is never mistaken for a finished one.
- Any required trailer (e.g. `Co-Authored-By: Alberto Gonzalez`) stays in the
  body. This convention governs the **subject line**; it does not remove the
  trailer.

## Alternatives Considered

| Alternative | Reason not chosen |
|---|---|
| Free-form messages | No structure and no link from a commit to the spec/task it serves; defeats the SDD traceability ADR-0004 is built for. |
| Conventional Commits without the `(spec: …)` reference | Gains a typed history but loses the spec linkage, which is the specific value here. |
| One squashed commit per feature | Loses the per-task granularity that the plan's numbered tasks and the TDD red/green cycle produce. |

## Consequences

- **Positive:** History is navigable by type and traceable to the spec each change
  implements; per-task commits map one-to-one to plan tasks.
- **Negative:** Requires discipline and is not machine-enforced (no commit hook
  yet — adding one would be a separate decision).
- **Note:** Coexists with, and does not supersede, any required commit trailer.

## References

- ADR-0003 — Adopt Spec-Driven Development
- ADR-0004 — Spec → Plan → Implement → Verify workflow (this is its Phase 4)
- ADR-0005 — Test-first development (the failing-test commit)
