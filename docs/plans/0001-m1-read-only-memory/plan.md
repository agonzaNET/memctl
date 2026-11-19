# Stage plan 0001 — M1 read-only memory

Stage: design milestone M1. Specs: SPEC-0001 (note parser), SPEC-0002
(in-memory index), SPEC-0003 (notes root and freshness check), SPEC-0004
(`memctl mcp` and the read tools), SPEC-0005 (`memctl check`). All Accepted.
This plan replaces `docs/specs/0001-note-parser/plan.md` (ADR-0018).

Exit of M1 (design): search and graph walk work over a copy of an existing
notes root of a few hundred notes, and edits made in an editor show up on the
next call.

## Slices

The tasks are ordered as four slices, each ending in something that runs:

| Slice | Tasks | Ends with |
|---|---|---|
| A — First tool call | T1–T6 | Claude Code reads a note through `memctl mcp` |
| B — Structure and search | T7–T10 | `search` ranks notes; observations and relations are parsed |
| C — Graph and listing | T11–T12 | `context` and `list` answer; `note_read` shows resolved links |
| D — Completeness | T13–T18 | `.gitignore`, path safety, all parser problems, `memctl check`, robustness, live check |

No spec is deferred.

## File map

| File | Change | Why |
|---|---|---|
| `go.mod`, `go.sum` | modify | Require the MCP SDK and `go.yaml.in/yaml/v3` (ADR-0011) |
| `internal/arch/imports_test.go` | modify | Map both modules to their owner packages (ADR-0010) |
| `cmd/memctl/main.go` | create | Subcommands, flags, environment, exit statuses, version |
| `cmd/memctl/log.go` | create | JSON logger on stderr |
| `cmd/memctl/lifecycle.go` | create | Shutdown context: stdin EOF, SIGINT, SIGTERM |
| `cmd/memctl/check.go` | create | `memctl check` report and exit status |
| `cmd/memctl/*_test.go` | create | Command-line tests against the built binary; end-to-end MCP test |
| `internal/parse/` | create | `note.go`, `parse.go`, `frontmatter.go`, `fence.go`, `heading.go`, `links.go`, `items.go`, `unsupported.go`, tests, `fuzz_test.go`, `testdata/` |
| `internal/index/` | create | `index.go` (storage), `resolve.go`, `tokenize.go`, `search.go`, `snippet.go`, `graph.go`, `list.go`, `links.go`, tests, benchmarks |
| `internal/note/` | create | `service.go` (Open, mutex, Call), `walk.go`, `load.go`, `fresh.go`, `gitignore.go`, `safepath.go`, tests, benchmarks |
| `internal/gitlog/repo.go` | create | Repository test shared with M2 (SPEC-0005 R7) |
| `internal/mcpserver/` | create | `server.go`, `result.go` (shape, codes), `log.go` (tool-call line), `tools.go` (descriptions, schemas between the agent-facing markers), `read.go`, `search.go`, `context.go`, `list.go`, `render.go`, tests, golden files |
| `CHANGELOG.md` | modify | `Added` entries in the tasks that make something visible: T4, T6, T10, T12, T16 |
| `docs/design/YYYY-MM-DD-m1-live-check.md` | create (T18) | Verification record of the live check |
| `docs/specs/0001-…/spec.md` to `0005-…/spec.md`, `docs/specs/README.md` | modify | Acceptance boxes, status `Implemented` once each spec is complete |

## Dependency order

```
T1 → T2 → T3 → T4 → T5 → T6      (slice A)
T6 → T7 → T8 → T9 → T10               (slice B)
T10 → T11 → T12                        (slice C)
T12 → T13, T14, T15 → T16 → T17 → T18  (slice D)
```

T13, T14 and T15 are independent of each other.

## Tasks

Each task: write the tests named after the requirements, run them and see them
fail for the expected reason, commit them
(`test(<pkg>): … (spec: SPEC-NNNN)`), implement, `make verify`, commit
(`feat(<pkg>): … (spec: SPEC-NNNN)`). A task that touches two specs names both
in the commit.

### Slice A — First tool call

- [x] **T1 — Parser core.** Require `go.yaml.in/yaml/v3` and map it to
  `internal/parse` in the import test. Types, line loop (BOM, CRLF, line numbers,
  `invalid_utf8`), frontmatter (delimiters, yaml.v3, known fields, raw
  key/value pairs, `bad_field`, `bad_frontmatter`, unclosed block), body,
  file-stem title, problem sorting.
  *SPEC-0001 R1–R8, R21, R22, R23, R24.*
- [x] **T2 — Notes root and load.** `note.Open` with root resolution, the walk
  (`.md` files, dot-entries skipped, no symlinks, no `.gitignore` yet), size
  limit, unreadable files, parallel load with SHA-256, the `notes loaded` line.
  *SPEC-0003 R1–R4, R6–R9.*
- [x] **T3 — Index storage and resolution.** (Done before T2: the startup
  load of T2 fills the index.) `Put`, `Remove`, `Get`, `Titled`,
  `Resolve` with the four steps, ambiguity, `Stale`.
  *SPEC-0002 R1, R3–R6.*
- [x] **T4 — Command line.** `memctl`, `memctl version`, `memctl mcp` flags and
  environment, usage errors (status 2), root errors (status 1), JSON logger on
  stderr, `started`/`stopped`, shutdown context on stdin EOF and signals.
  CHANGELOG: `memctl version`.
  *SPEC-0004 R1–R4, R6, R26.*
- [x] **T5 — Freshness check.** Stat walk, rehash on size or time change,
  meta-only update when the hash is equal, add/remove, the service mutex and
  `Call` wrapper used by every tool.
  *SPEC-0003 R10, R12, R13.*
- [x] **T6 — MCP server and `note_read`.** Require the MCP SDK and map it to
  `internal/mcpserver` in the import test. SDK server over stdio, server info
  and instructions, result shape and fixed error codes, `ref` resolution
  errors, tool-call log line, schema and annotations, agent-facing markers in
  `tools.go`, `note_read` with metadata, fenced content, `section` and
  `sha256` (`section` moves to T7, which parses headings; the links list
  fills in as T8 parses links). End-to-end test
  over pipes. Manual check: `claude mcp add` reads a note.
  CHANGELOG: `memctl mcp` with `note_read`.
  *SPEC-0004 R5, R7–R17, R18; SPEC-0003 R12 (tool calls under the mutex).*

### Slice B — Structure and search

- [x] **T7 — Fences, headings, sections.** Fence rules, ATX headings, the
  Observations and Relations sections; `note_read` with `section`.
  *SPEC-0001 R9–R12; SPEC-0004 R16.*
- [ ] **T8 — Links, observations, relations.** Link scanner with code spans,
  `bad_link`, observation and relation forms, links inside observations.
  *SPEC-0001 R13–R18.*
- [ ] **T9 — Search.** Tokenizer, five fields, BM25F, OR semantics, filters,
  limit, `ErrEmptyQuery`, snippets.
  *SPEC-0002 R7–R14.*
- [ ] **T10 — `search` tool.** Rendering, folder normalisation, errors, log
  counts. CHANGELOG: `search`.
  *SPEC-0004 R19–R21.*

### Slice C — Graph and listing

- [ ] **T11 — Graph, listing, link report.** Edges and backlinks, breadth-first
  walk with depth, `relTypes`, unresolved edges of the start, self-links,
  200-note cap, `ErrNotFound`; `List`; `Links()`.
  *SPEC-0002 R15–R21.*
- [ ] **T12 — `context` and `list` tools; links in `note_read`.** Rendering,
  truncation lines, 500-entry cap, log counts; `note_read`'s `Links` list with
  resolved, ambiguous, dangling and stale targets. CHANGELOG: `context`, `list`.
  *SPEC-0004 R15 (links list), R22–R25.*

### Slice D — Completeness

- [ ] **T13 — `.gitignore` matcher.** Pattern rules, nested files, precedence,
  reload on change, comparison with `git check-ignore` when git is present.
  *SPEC-0003 R5, R11.*
- [ ] **T14 — Path safety.** `note.SafePath` and its rejections, including a
  symlinked folder. (Used by M2; specified in M1.)
  *SPEC-0003 R14, R15.*
- [ ] **T15 — Parser problems.** Lines outside their section; setext headings,
  HTML blocks, indented code, once per block, none in fences.
  *SPEC-0001 R19, R20.*
- [ ] **T16 — `memctl check`.** Findings from parser problems, link report,
  skipped files and the git test (`internal/gitlog/repo.go`); severities;
  format, sorting, summary; exit statuses; read-only proof. CHANGELOG:
  `memctl check`.
  *SPEC-0005 R1–R8.*
- [ ] **T17 — Robustness and performance.** `FuzzParse` with seed corpus;
  determinism test; index property test (random `Put`/`Remove`); freshness
  property test (random file operations); `-race` on load and concurrent calls;
  the design's example note golden test; benchmarks `BenchmarkBuild`,
  `BenchmarkSearch`, `BenchmarkWalk`, `BenchmarkOpen`, `BenchmarkCheck` with
  5,000 synthetic notes, results recorded below.
  *SPEC-0001 R1, R2, R22, R24; SPEC-0002 R2, R22; SPEC-0003 R8, R12, R13, R16.*
- [ ] **T18 — Live check and close.** Build with `make build`; run against a
  copy of an existing notes root of a few hundred notes from Claude Code: all
  four tools, an editor edit seen on the next call, `memctl check` on the same
  copy. Write the verification record under `docs/design/`. Tick every
  acceptance box of SPEC-0001 to SPEC-0005 and set each to `Implemented` with
  its index row; tick this plan. Commit
  `docs: M1 implemented (spec: SPEC-0001..0005)`.

## Requirement coverage

| Spec | Requirements | Tasks |
|---|---|---|
| SPEC-0001 | R1–R8, R21–R24 | T1 (R1, R2, R22, R24 again in T17) |
| | R9–R12 | T7 |
| | R13–R18 | T8 |
| | R19, R20 | T15 |
| SPEC-0002 | R1, R3–R6 | T3 |
| | R7–R14 | T9 |
| | R15–R21 | T11 |
| | R2, R22 | T17 |
| SPEC-0003 | R1–R4, R6–R9 | T2 |
| | R10, R12, R13 | T5 (R12, R13 again in T17) |
| | R5, R11 | T13 |
| | R14, R15 | T14 |
| | R16 | T17 |
| SPEC-0004 | R1–R4, R6, R26 | T4 |
| | R5, R7–R15, R17, R18 | T6 (R15 completed in T12) |
| | R16 | T7 |
| | R19–R21 | T10 |
| | R22–R25 | T12 |
| SPEC-0005 | R1–R8 | T16 |

## Definition of done

- `make verify` passes: `go test ./...` (fuzz seed corpus, import scan, golden
  files, end-to-end tests), `go vet ./...`, `gofmt -l .` empty, `check-docs.sh`
  including the agent-facing text check of `tools.go`.
- `go test -race ./...` passes.
- `go test -run '^$' -fuzz FuzzParse -fuzztime 60s ./internal/parse` finds no
  failure.
- Every requirement of SPEC-0001 to SPEC-0005 has at least one passing test
  whose name carries it (`SPEC0001_R7` style); every acceptance box is ticked;
  the five specs and their index rows say `Implemented`.
- The live check is recorded under `docs/design/` and listed in its README.
- `CHANGELOG.md` `[Unreleased]` lists `memctl version`, `memctl mcp` with the
  four read tools, and `memctl check`.

## Verification notes

Benchmark results (SPEC-0002 R22, SPEC-0003 R16), filled in by T17:

| Benchmark | Target | Result |
|---|---|---|
| Build index, 5,000 notes | < 2 s | |
| Search, 5,000 notes | < 20 ms | |
| Walk depth 3, 5,000 notes | < 20 ms | |
| Startup load, 5,000 notes | < 2 s | |
| Freshness check, no change | < 30 ms | |

## Notes

- The dev VM has Go 1.26.8 in `~/sdk/go` (not on `PATH`); commands run with
  `PATH=$HOME/sdk/go/bin:$PATH`.
- Commits are local on `main`; the maintainer pushes.
