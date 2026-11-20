# Stage plan 0005 — Repeated live-check fixes (v0.2.0)

Stage: the fixes of findings N2 to N4 of the repeated live check of
2026-11-20
([`2026-11-20-live-check-repeat.md`](../../design/2026-11-20-live-check-repeat.md)),
which the maintainer chose to make in v0.2.0. Specs, amended on 2026-11-20 and
Accepted: SPEC-0002 (R14, R27–R29) and SPEC-0004 (R19, R22, R28, R31). No
spec is deferred.

Exit: every amended or added requirement has a passing test that names it,
and a short live check by the maintainer on a copy of a real notes root shows
the three changes; then v0.2.0 is released.

## Slices

| Slice | Tasks | Ends with |
|---|---|---|
| A — Snippet around the match (N2) | T1–T2 | A `search` hit on a long line shows the matched term, with `…` on each cut side |
| B — `context` edge scope (N3) | T3–T4 | `context` with `edges` `start` lists the same notes with only the start note's edges; the description states the size |
| C — Near names in suggestions (N4) | T5–T6 | An unknown ref `cisco` suggests a note that holds `cisco1`; `search` is unchanged |
| D — Close and release | T7–T9 | Benchmarks, CHANGELOG and README; live check; specs `Implemented`; v0.2.0 tagged |

## File map

| File | Change | Why |
|---|---|---|
| `internal/index/snippet.go` | create | Snippet choice and the new window, moved out of `search.go` (ADR-0002) (SPEC-0002 R14, R27) |
| `internal/index/search.go` | modify | Snippet code moved out; `Query.NearNames` and the term expansion (SPEC-0002 R29) |
| `internal/index/graph.go` | modify | `EdgeScope` (`EdgesAll`, `EdgesStart`) and the `Walk` parameter (SPEC-0002 R28) |
| `internal/index/search_test.go`, `graph_test.go` | modify | Tests named `SPEC0002_R27`, `R28`, `R29` |
| `internal/index/bench_test.go`, `property_test.go` | modify | New `Walk` signature |
| `internal/mcpserver/context.go` | modify | `edges` input, scope, `Edges of the start note:` heading (SPEC-0004 R22, R31) |
| `internal/mcpserver/render.go` | modify | `NearNames` in the not-found suggestions (SPEC-0004 R28) |
| `internal/mcpserver/tools.go` | modify | `search` description (snippet cut); `context` description and schema (`edges`, size sentence) (SPEC-0004 R19, R22, R31) |
| `internal/mcpserver/*_test.go`, `testdata/` | modify | Tests named `SPEC0004_R19`, `R22`, `R28`, `R31`; golden files |
| `CHANGELOG.md` | modify (T7) | `[Unreleased]`: `Added` for `edges`, `Changed` for snippets and suggestions |
| `README.md` | modify (T7) | Tool table rows of `search` and `context` |
| `docs/design/YYYY-MM-DD-v0.2.0-live-check.md`, `docs/design/README.md` | create, modify (T8) | Verification record of the live check |
| Specs and `docs/specs/README.md` | modify (T8) | Acceptance boxes; `Implemented` |
| `CHANGELOG.md`, `docs/design/YYYY-MM-DD-v0.2.0-release.md` | modify, create (T9) | Release section and release record (ADR-0012) |

Interface changes inside `internal/index`:
`Walk(start string, depth int, relTypes []string, scope EdgeScope)`, with
`EdgesAll` the zero value; `Query` gains `NearNames bool`. Only
`internal/mcpserver` and the index's own tests call them.

## Dependency order

```
T1 → T2                       (slice A)
T3 → T4                       (slice B)
T5 → T6                       (slice C)
all → T7 → T8 → T9            (slice D)
```

Slices A, B and C are independent of each other.

## Tasks

Each task: write the tests named after the requirements, run them and see them
fail for the expected reason, commit them (`test(<pkg>): failing test … (spec:
SPEC-NNNN)`), implement, `make verify`, commit (`feat(<pkg>): …` or
`fix(<pkg>): … (spec: SPEC-NNNN)`).

### Slice A — Snippet around the match (N2)

- [x] **T1 — Snippet window.** Move the snippet code to `snippet.go` without a
  behaviour change (its own commit, existing tests green). Then: find the
  first token of the chosen line equal to a query term (earliest rune, then
  longest), by rune offsets of the trimmed line; window of 200 runes
  centred on it, clamped to the line; a line without a query term starts at
  rune 0; trim spaces at a cut edge; `…` on each cut side; lines of 200 runes
  or fewer whole. The table of the R14, R27 acceptance box.
  *SPEC-0002 R14, R27.*
- [x] **T2 — `search` output.** Golden file with a hit on a long line (`…` on
  both sides); the description says that a long line is cut around the first
  matching word.
  *SPEC-0004 R19.*

### Slice B — `context` edge scope (N3)

- [x] **T3 — Walk edge scope.** `EdgeScope` with `EdgesAll` (zero value) and
  `EdgesStart`; with `EdgesStart`, keep only the collected edges whose source
  or target is the start; nodes, distances and `Truncated` unchanged;
  `relTypes` in both scopes. Update the callers to pass `EdgesAll`.
  *SPEC-0002 R28.*
- [x] **T4 — `edges` input.** Schema property `edges` with `enum`
  `["all", "start"]` and default `all`; map to the scope; heading
  `Edges of the start note:`; description: both values, the same notes with
  only the start note's connections, the size sentence for depth 2 or 3, and
  the edge rule "unless `edges` is `start`". Golden files at depth 2 with and
  without `edges`; `edges` `other` refused by the SDK.
  *SPEC-0004 R22, R31.*

### Slice C — Near names in suggestions (N4)

- [x] **T5 — `NearNames`.** For each query term of one part, all letters, at
  least two runes: add every indexed token that is the term followed by one or
  more digits as one more query term (scoring, hit test, snippet). A scan of
  the postings' keys is enough: it runs only for a `not_found` error. The
  fixture of the R29 acceptance box.
  *SPEC-0002 R29.*
- [x] **T6 — Suggestions use near names.** `notFound` sets `NearNames`;
  `search` does not. Test: unknown ref `cisco` lists the `cisco1` note;
  `search` for `cisco` on the same root finds no notes; no ref text logged.
  *SPEC-0004 R28.*

### Slice D — Close and release

- [x] **T7 — Benchmarks, changelog, README.** Rerun `BenchmarkSearch`,
  `BenchmarkWalk` and `BenchmarkWalkCold` with the commands of stage plan 0004
  and record them below. `CHANGELOG.md` `[Unreleased]`: `Added` — the
  `context` input `edges`; `Changed` — snippets centred on the match, near
  names in `not_found` suggestions. README tool table: `search` snippets,
  `context` `edges`.
  *SPEC-0002 R22 (check only).*
- [ ] **T8 — Live check and close.** `make build GOOS=darwin GOARCH=arm64`
  for the maintainer's Mac. The maintainer runs, with Claude Desktop on a copy
  of the real notes root:
  1. `search` for a word that sits after the first 200 characters of a long
     line: the snippet shows the word, with `…` where the line was cut;
  2. `context` at depth 2 on the note of check R1 of the repeated live check,
     once without `edges` and once with `edges` `start`: the same first line
     and notes; the second lists only the start note's edges under
     `Edges of the start note:`;
  3. `note_read` of an unknown ref whose near name exists (such as `cisco`
     for `cisco1`): `not_found` names that note; `search` for the same word
     does not find it.

  Write `docs/design/<date>-v0.2.0-live-check.md` and list it in
  `docs/design/README.md`; tick every acceptance box; set SPEC-0002 and
  SPEC-0004 to `Implemented` with their index rows; tick this plan. Commit
  `docs: repeated live-check fixes implemented (spec: SPEC-0002, SPEC-0004)`.
- [ ] **T9 — Release 0.2.0.** The steps of `docs/agents/environment.md`
  "Releasing": rename `[Unreleased]` to `[0.2.0] - <date>`, open a fresh
  `[Unreleased]` with the comment, update the link references (compare
  `v0.1.0...v0.2.0`), `make verify`, commit, `git tag -a v0.2.0 -m "memctl
  0.2.0"`. The maintainer pushes and runs
  `gh workflow run release.yml -f tag=v0.2.0`. Verify the downloaded assets
  against `SHA256SUMS`, run one on a fresh root, write
  `docs/design/<date>-v0.2.0-release.md` and list it in the design README.
  *ADR-0012; no requirement.*

## Requirement coverage

| Spec | Requirements | Tasks |
|---|---|---|
| SPEC-0002 | R14, R27 | T1 |
| SPEC-0002 | R28 | T3 |
| SPEC-0002 | R29 | T5 |
| SPEC-0002 | R22 (rerun only) | T7 |
| SPEC-0004 | R19 | T2 |
| SPEC-0004 | R22, R31 | T4 |
| SPEC-0004 | R28 | T6 |
| SPEC-0004 | Live check (v0.2.0) | T8 |

## Definition of done

- `make verify` passes, and `govulncheck ./...` reports no vulnerability the
  code calls.
- Every amended or added requirement has at least one passing test whose name
  carries it; every acceptance box of SPEC-0002 and SPEC-0004 is ticked; both
  specs and their index rows say `Implemented`.
- The live check is recorded under `docs/design/` and listed in its README.
- `CHANGELOG.md` holds the `[0.2.0]` section, and the release record is
  written.

## Verification notes

Benchmark results (SPEC-0002 R22) after T1–T6, on the same development VM
(linux/arm64) and with the same commands as stage plan 0004
(`go test -run '^$' -bench . -benchtime 5x`, 10x for the walks), 5,000
synthetic notes:

| Benchmark | Target | Stage plan 0004 | Now |
|---|---|---|---|
| Build index (`BenchmarkBuild`) | < 2 s | 0.59 s | 0.62 s |
| Search (`BenchmarkSearch`) | < 20 ms | 1.5 ms | 1.6 ms |
| Walk depth 3, index unchanged (`BenchmarkWalk`) | < 20 ms | 12.5 ms | 10.6 ms |
| Walk depth 3, first walk after a change (`BenchmarkWalkCold`) | < 20 ms | 105 ms — missed | 100 ms — missed |

No change beyond the noise of the VM: the snippet window works on the shown
hits only, the edge scope filters edges already collected, and near names
run only for a `not_found` error. The first walk after a change still misses
its target, as before.
