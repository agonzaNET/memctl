# workflow.md — Development Workflow (SDD)

How a change goes from idea to shipped code in this repo. Read this before
starting any non-trivial change. The decisions behind it are recorded in
ADR-0003 (SDD), ADR-0004 (this workflow) as amended by ADR-0018 (specs first,
stage plans), ADR-0005 (test-first), ADR-0006 (commits), and ADR-0007
(changelog) — check their status in the
[ADR index](../adr/README.md#index) before treating any rule here as settled.

Verification means tests pass, the linter is clean, the documentation
consistency check (`scripts/check-docs.sh`) passes, and a sample run of the
changed behaviour (a tool call from an MCP client, or `memctl check` on a notes
root) shows the expected result. `make verify` runs all of these but the
sample run.

---

## Core principle

Think before you code. Every non-trivial task follows **Spec → Plan → Implement
→ Verify**. Do not jump straight to implementation. When in doubt, write a spec
first.

## When the full workflow applies

Use Spec → Plan → Implement → Verify when a change:

- Touches more than two files, or
- Cannot be described in one sentence, or
- Touches a security boundary, configuration, a public interface/contract, or a
  data shape, or
- Is expected to take more than ~30 minutes.

Skip the formal workflow and implement directly for typos, renames, log lines,
single-variable changes, and obvious one-file diffs.

### Build and release artifacts are ADR-governed, not spec-governed

Build, packaging and CI artifacts that contain **no application code** and have
**no behaviour testable by the Go suite** — e.g. the `Makefile`, a GitHub
Actions workflow, `.mise.toml`, `scripts/check-docs.sh` —
do **not** get a `docs/specs/` spec, even when they trip the triggers above
(touching configuration or a security boundary, or spanning several files). A
spec here would be hollow: there are no `R`-numbered requirements a failing Go
test can prove, so the Spec → test loop has nothing to grip.

These changes are instead **governed by an ADR** (per [`adr.md`](adr.md): "draft
an ADR when you change the project structure or deployment model"). The process
is lighter but not absent — for a non-trivial ops artifact you still:

1. **Agree the approach first.** Record the decision in an ADR and get human
   sign-off on it (an ADR in `Proposed` state, or a short written plan) *before*
   writing the artifact — this preserves the "stop and wait for review"
   checkpoint without the spec ceremony.
2. **Keep the docs in sync.** Add a `CHANGELOG.md` entry for the
   user-visible change and ensure `scripts/check-docs.sh` stays green (the
   ADR must appear in the [ADR index](../adr/README.md#index) with a matching
   Status and Area).

This carve-out is narrow: anything that changes Go code — including code that
*reads* a new flag or environment variable — stays fully spec-governed. The
build targets and the release workflow (ADR-0012) are instances of this rule.

---

## Branching — work on `main` by default

Commit directly to `main`. Do **not** create a branch for a change unless the
human explicitly asks for one (or for a PR). The project has a single
maintainer: the SDD checkpoints (spec review, plan review, Phase 4
verification) already provide the review gate a PR would, and branches only
add merge overhead. If your team reviews through PRs, replace this section.

- Commit each task to `main` as Phase 3 describes.
- Open a PR only when you were asked to work on a branch. The PR step of
  ADR-0004 ("open a PR only when all tasks are checked off") applies only in
  that case.
- The one standing exception is the worktree a subagent runs in under
  [Parallel work](#parallel-work-only-if-using-subagentsworktrees): that is an
  explicit instruction to work off `main`, and its result still lands on `main`.

---

## Phase 1 — Spec (what)

Before writing code, create or update the capability spec under `docs/specs/`.
Each capability gets its own folder, `NNNN-slug/` (sequential index, never
reused — see [`specs/README.md`](../specs/README.md)), and the spec itself is
`docs/specs/NNNN-slug/spec.md`. Use
[`SPEC-TEMPLATE.md`](../specs/SPEC-TEMPLATE.md): Purpose, Scope (in/out), Inputs,
Outputs, numbered testable **Requirements** (MUST/MUST NOT/SHOULD), Error
handling, and Acceptance criteria ("done looks like"). Link the governing ADR(s).

Draft the spec in **plan mode** — do not create code or edit implementation files
in this phase. When the spec is written, **stop and wait for human review**.

**Specs first (ADR-0018).** Write and get acceptance for every spec of the
planned work before any implementation starts — for `memctl`, every spec of a
milestone before its first line of code. Specs are reviewed one at a time,
planned once the stage's specs are all accepted. The one exception is a spec
that cannot be written without code that does not exist yet (it depends on
measured behaviour, or on an interface the earlier code settles): note it in
the stage plan as deferred, and write it once that code is implemented.

> If the change modifies an existing capability, edit its spec in the same change
> rather than writing a new one. When behaviour and spec disagree, one is a bug.

## Phase 2 — Plan (how)

After the specs of a stage are accepted, write its plan (ADR-0018):

- **Stage plan** — when several specs only deliver value together (a
  milestone), one plan for all of them: `docs/plans/NNNN-slug/plan.md`, with its
  own sequential index. Order the tasks as thin end-to-end slices, so a working
  result (for example the first tool call) arrives early, and name each task's
  requirements as `SPEC-NNNN Rn`.
- **Spec plan** — when a spec stands alone, `docs/specs/NNNN-slug/plan.md`,
  alongside the spec, with requirements named `Rn`.

A plan contains:

1. **File map** — every file created or modified, and why.
2. **Dependency order** — which tasks block which.
3. **Numbered task list** — atomic checkbox tasks, each completable in one
   session. Each task names the spec requirement(s) it implements (e.g. `T2 —
   reject unknown flags (R3)`, or `SPEC-0002 R3` in a stage plan), so every
   requirement maps to at least one task.
4. **Definition of done** — the build/test command that proves the feature
   complete, plus the rule that every requirement has a passing test that names it
   and every acceptance-criteria box in the spec is checked.

Stay in **plan mode**; write no implementation code. **Stop and wait for human
review** of the plan before implementing. Plan files are committed and kept as an
audit trail — do not delete them after implementation. A spec plan that a
stage plan takes over stays in place with a note at its top naming the stage
plan.

## Phase 3 — Implement (task by task, test-first)

Work the plan one numbered task at a time. Per ADR-0005:

1. Write the test, naming it after the spec requirement it proves (e.g. a test
   for `R3`), so each requirement traces to the test that enforces it.
2. Run it; confirm it fails for the expected reason.
3. Commit the failing test (see Phase 4 for the message).
4. Implement until it passes.
5. Do **not** modify the test to make it pass — fix the implementation.
6. Run the full suite after the task; fix regressions before the next task.

Commit one task before starting the next. If you discover the plan is wrong,
**stop and surface the conflict** — do not silently deviate. Test style
(table-driven, stdlib `testing`, no assertion libraries) is in [`backend.md`](backend.md).

## Phase 4 — Verify & commit

Before declaring a task or feature done:

1. Run the full test suite (`go test ./...`) — paste the output as evidence.
2. Run the linter / type-checker (`go vet ./... && gofmt -l .`) — fix every
   error. Then run `scripts/check-docs.sh` and fix any mismatch it reports: it
   asserts every spec's and ADR's Status matches its index row, so a forgotten
   index update fails the build instead of drifting silently. If your change
   edits `check-docs.sh` itself, also run its regression test
   (`scripts/check-docs.test.sh`), confirm every case passes, and add a case
   for the behaviour you changed — the gate is only trustworthy if its own
   logic is tested, and the
   test exists to catch the silent false negative (the script printing PASS
   while the docs have actually drifted). `make verify` runs every check in
   this step.
3. When you add or change a tool, update its description, input schema and
   error codes in `internal/mcpserver/tools.go` in the same change, and the
   tool table in the design if the tool's inputs or behaviour changed (see
   [`tools.md`](tools.md)).
4. Confirm every spec requirement (`R1`, `R2`, …) has a passing test that names
   it, and tick each acceptance-criteria box in the spec against the test or run
   that proves it. A requirement with no passing test is not done.
   Once every acceptance-criteria box is ticked, set the spec's **Status** to
   `Implemented` and update the matching row in
   [`../specs/README.md`](../specs/README.md) so the index agrees with the spec
   file. (Status vocabulary and lifecycle are in
   [`../specs/SPEC-TEMPLATE.md`](../specs/SPEC-TEMPLATE.md).)
5. Update `CHANGELOG.md`'s `[Unreleased]` section with a human-readable entry for
   any user-visible change (ADR-0007 — see
   [`changelog.md`](changelog.md) for the format and categories). Skip this only
   for changes nobody outside the code can observe (pure refactors, internal
   test-only changes).
6. Commit with a spec-referencing Conventional Commit (ADR-0006):
   `feat(scope): short description (spec: SPEC-NNNN)`. When the repo or the
   harness requires a trailer (for example a `Co-Authored-By:` line), keep
   it in the body; ADR-0006 makes no trailer mandatory by itself.
7. Commit to `main` — see [Branching](#branching--work-on-main-by-default).
   If the human asked for a branch, open the PR only when all plan tasks are
   checked off and the build is green.

Never assert "it works" — show evidence (test output, linter output, a sample
run of the changed behaviour).

Never push. Commits stay local; the maintainer pushes and pushes tags.

### Quality evals for behaviour a model judges

Unit tests prove that the code does what the spec says. They do not prove that
a ranking, a retrieval or a tool description is *good enough* for the model
that uses it. `memctl` has such behaviour: the BM25F ranking of `search` and
the tool descriptions. When it is tuned, add an **eval**: a committed set of
realistic queries with the expected notes, a harness that runs them against
`memctl` over a fixed notes snapshot and scores the result, and committed
result files. The rules that make it useful:

- **It tests `memctl`, not the data.** Run it before and after a change to
  the behaviour it scores, on the same input (the same corpus commit), and
  gate the release on the comparison: no case gets worse, the overall score
  does not drop. An absolute threshold drifts as the data changes.
- **Its cases are kept current.** Each case declares how it is scored (for
  example "any of these answers" or "all of them"); the harness checks the
  case set against the data before a run, and a written policy keeps the set
  balanced.
- **It has its own tests** (the harness's unit tests, a `make` target), and its
  results are committed next to it, named by version or change.

An eval is a spec-level decision: record it in the design or an ADR, and its
harness in a spec, as for any capability.

---

## Files this workflow produces

```
docs/specs/
  NNNN-slug/            # one folder per capability
    spec.md             # spec — what the capability must do (permanent contract)
    plan.md             # plan of a standalone spec (kept as audit trail)
    ...                 # any supporting material for this spec (diagrams, notes)
docs/plans/
  NNNN-slug/            # one folder per stage (ADR-0018)
    plan.md             # stage plan — file map + tasks naming SPEC-NNNN Rn
```

It also **updates** (never recreates) `CHANGELOG.md` at the repo root: each
consumer-visible change adds a line under `[Unreleased]` in the same commit (see
Phase 4 and [`changelog.md`](changelog.md)).

## Anti-patterns

- Writing code for a stage before all of its specs are accepted, outside the
  deferred-spec exception (ADR-0018).
- Starting to code before a spec exists for a non-trivial change.
- Modifying a test to make it pass instead of fixing the code.
- Silently deviating from the plan — surface the conflict first.
- Combining multiple plan tasks into one commit.
- Claiming success without running the actual verification command.
- Checking off an acceptance criterion (or a requirement) that no test or observed
  run actually proves.
- Leaving a TODO comment as a substitute for a follow-up task in the plan.
- Shipping a stub/placeholder and calling the task done.
- Shipping a consumer-visible change without a `CHANGELOG.md` entry under
  `[Unreleased]`.
- Creating a branch, or opening a PR, that nobody asked for.

## Parallel work (only if using subagents/worktrees)

- Each subagent takes exactly one task from the plan.
- The orchestrating session owns the plan file and updates the checkboxes.
- A fresh-context reviewer session checks completed tasks — it did not write the
  code, so it is not biased toward it.
