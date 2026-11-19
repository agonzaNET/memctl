# Plan — SPEC-0001 Note parser

> **Replaced by the M1 stage plan** under `docs/plans/` (ADR-0018), once the
> v1 specs (M1 and M2) are accepted. Kept as history; do not implement from it.

Spec: [`spec.md`](spec.md) (Accepted). Governing ADRs: ADR-0002, ADR-0005,
ADR-0010, ADR-0011.

## File map

| File | Change | Why |
|---|---|---|
| `go.mod`, `go.sum` | modify | Require `gopkg.in/yaml.v3` (ADR-0011) |
| `internal/arch/imports_test.go` | modify | Allow `gopkg.in/yaml.v3` → `internal/parse` (R23, ADR-0010) |
| `internal/parse/doc.go` | create | Package comment: what the parser does and does not do |
| `internal/parse/note.go` | create | Types: `Note`, `Heading`, `Observation`, `Relation`, `Link`, `Problem`, `Field`; problem-code constants |
| `internal/parse/parse.go` | create | `Parse(rel string, data []byte) Note`: BOM, line splitting, UTF-8 check, the line loop that drives the other files, problem sorting |
| `internal/parse/frontmatter.go` | create | Delimiters, yaml.v3 decoding, known fields, raw key/value pairs, title fallback (R1–R7) |
| `internal/parse/fence.go` | create | Fence open/close state (R9, R10) |
| `internal/parse/heading.go` | create | ATX headings and the two sections (R11, R12) |
| `internal/parse/links.go` | create | `[[link]]` scanning with code spans (R17, R18) |
| `internal/parse/items.go` | create | Observation and relation line forms, the outside-section problems (R13–R16, R19) |
| `internal/parse/unsupported.go` | create | Setext, HTML block and indented-code detection (R20) |
| `internal/parse/*_test.go` | create | One test file per source file, table-driven, test names carry the requirement |
| `internal/parse/fuzz_test.go` | create | `FuzzParse` with a seed corpus (R22, R24) |
| `internal/parse/testdata/` | create | The design's example note and a few real-world-shaped notes |
| `docs/specs/README.md`, `spec.md` | modify (T11) | Status `Implemented`, acceptance boxes ticked |

No `CHANGELOG.md` entry: the parser is not observable by a user until `memctl
check` or the MCP tools ship (SPEC-0004, SPEC-0005).

## Dependency order

T1 → T2 → T3 → T4 → T5 → T6 → T7 → T8 → T9 → T10 → T11.

T1 sets up the types and the line loop every later task hooks into. T3
(fences) comes before everything that must ignore fenced lines. T5 (links)
comes before T6 and T7, which reuse the link scanner. T8 depends on the line
forms of T6 and T7.

## Tasks

Each task: write the tests, run them and see them fail for the expected
reason, commit them (`test(parse): … (spec: SPEC-0001)`), implement, run
`make verify`, commit (`feat(parse): … (spec: SPEC-0001)`).

- [ ] **T1 — Types and line loop (R8, R21, R22 partial, R24 partial).**
  Types and problem codes; `Parse` strips one BOM, splits on `\n` and drops a
  trailing `\r`, numbers lines from 1, reports `invalid_utf8` once at the first
  bad line, returns a note for empty input. Problems sorted by line, then code.
- [ ] **T2 — Frontmatter (R1–R7, R23).** Add yaml.v3 to `go.mod` and to the
  import map. Delimiters `---` and `---`/`...` with trailing spaces; decode
  into a `yaml.Node`; read the known keys; `Frontmatter` as ordered
  key/raw-YAML pairs (no yaml.v3 type exported); `tags`/`aliases` as list or
  string, trimmed, de-duplicated; `created`/`updated` as RFC 3339 or a date;
  `bad_field`, `bad_frontmatter`, unclosed block; `Body`, `BodyLine`; file-stem
  title.
- [ ] **T3 — Fences (R9, R10).** Opening and closing rules for backticks and
  tildes, longer closing fence, info string, unclosed fence to end of file; the
  line loop skips fenced lines for every later recogniser.
- [ ] **T4 — Headings and sections (R11, R12).** ATX recognition with closing
  `#`s; `Headings`; section state for `## Observations` / `## Relations`
  (case-insensitive), ended by level 1 or 2, kept by deeper headings.
- [ ] **T5 — Links and code spans (R17, R18).** Scanner for `[[target]]` and
  `[[target|label]]`, several per line, trimmed, forbidden characters,
  `bad_link` for an empty target; backtick code spans of matching run length
  on one line hide links; unclosed runs are literal. Path targets kept as
  written. Links of body lines recorded.
- [ ] **T6 — Observations (R13–R15).** The `- [category] text` form under the
  Observations section; rejected forms (task items, one-letter or upper-case
  category, indented, `*`/`+` bullets, empty text) give no observation and no
  problem; links in the text recorded via T5.
- [ ] **T7 — Relations (R16).** The `- type [[Target|label]]` form under the
  Relations section, trailing text ignored; the relation's link not in
  `Links`; other lines in the section scanned for links.
- [ ] **T8 — Lines outside their section (R19).** `observation_outside_section`
  and `relation_outside_section`, including a relation line under
  Observations and the reverse; their links recorded.
- [ ] **T9 — Unsupported forms (R20).** `setext_heading` (not after a blank
  line, a heading or frontmatter), `html_block`, `indented_code` (after a blank
  line, not in a list item), once per block at its first line; none inside
  fences.
- [ ] **T10 — Robustness and the design example (R1, R2, R22, R24).** Golden
  test of the design's example note against the values in the spec's
  "Outputs"; `FuzzParse` (no panic, deterministic, line numbers within the
  file) with a seed corpus from `testdata/`; a test that parses every testdata
  note twice and compares with `reflect.DeepEqual`.
- [ ] **T11 — Verify and close.** `make verify` green; every requirement R1–R24
  named by at least one passing test (`grep -o 'R[0-9]\+' internal/parse/*_test.go`
  covers R1–R24); tick each acceptance box; set the spec and its index row to
  `Implemented`; commit `docs(spec): SPEC-0001 implemented`.

## Definition of done

- `make verify` passes: `go test ./...` (including `FuzzParse`'s seed corpus
  and the import scan), `go vet ./...`, `gofmt -l .` empty, `check-docs.sh`.
- A short fuzz run, `go test -run '^$' -fuzz FuzzParse -fuzztime 60s ./internal/parse`,
  finds no failure.
- Every requirement R1–R24 has at least one passing test whose name carries it.
- Every acceptance-criteria box in the spec is ticked, and the spec and its
  index row say `Implemented`.

## Notes

- Go is not on the dev VM's `PATH` yet: a Go 1.26.8 toolchain was unpacked to
  `~/sdk/go` for this work (mise's installer host is blocked by the egress
  allowlist). `make verify` was run with it and passes.
- Commits are local on `main`; the maintainer pushes.
