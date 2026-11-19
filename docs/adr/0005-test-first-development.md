# ADR-0005 - Test-first development for non-trivial logic

## Status

Accepted

<!--
Status values: Proposed | Accepted | Rejected | Deprecated | Superseded by ADR-NNNN
-->

**Area:** Methodology

## Context

ADR-0003 adopted specs as testable behavioural contracts and noted that specs
can later seed automated tests. Without executable tests, a spec is enforced only
by prose and review — exactly the drift risk ADR-0003 flagged. Establishing a
test discipline from the start, while the suite is near-empty, is far cheaper
than retrofitting one onto a grown codebase.

The test *style* and tooling for this project's stack live in the relevant
`docs/agents/` guide; this ADR decides *when* tests are written.

## Decision

> For non-trivial logic, tests are written **before** the implementation (TDD).

The loop for each implementation task:

1. Write the test, naming it after the spec requirement it proves (e.g. `R3`).
2. Run it and confirm it fails for the expected reason.
3. Commit the failing test (see ADR-0006 for the message form).
4. Implement until the test passes.
5. Do **not** modify the test to make it pass — fix the implementation. A test
   only changes when the spec it encodes changes.
6. Run the full suite after the task and fix any regression before moving on.

"Non-trivial logic" means anything with branching or parsing — configuration
loading, auth/middleware, request handling, response shaping, and similar. Pure
renames, log lines, and trivial wiring are exempt. Test style stays exactly as
the stack guide specifies.

## Alternatives Considered

| Alternative | Reason not chosen |
|---|---|
| Tests optional | Specs stay unenforced by anything executable; the drift risk named in ADR-0003 is unaddressed. |
| Test-after (implement, then add tests) | Tests written after the fact tend to codify whatever the code already does — bugs included — and apply far less design pressure. |
| Rely on specs + manual review, no tests | Specs are prose and reviews are point-in-time; neither guards against regressions on later changes. |

## Consequences

- **Positive:** Specs gain executable enforcement; each capability spec can seed
  the test that proves it.
- **Positive:** Naming tests after the requirement they prove makes the
  spec → test mapping explicit and checkable; a requirement with no test is
  visibly unenforced.
- **Positive:** Design pressure is applied before the implementation exists, which
  tends to produce smaller, more testable units.
- **Negative:** Slower per task, and a discipline that has to be sustained.
  Mitigated by exempting trivial changes.
- **Note:** This sets the *when*; the *how* (style, tooling, assertion approach)
  lives in the stack guide under `docs/agents/`.

## References

- ADR-0003 — Adopt Spec-Driven Development (specs as testable contracts)
- ADR-0004 — Spec → Plan → Implement → Verify workflow (this is its Phase 3)
- ADR-0006 — Conventional commit messages (covers the failing-test commit)
