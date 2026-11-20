# Stage plan 0004 — Live-check fixes

Stage: the fixes the maintainer chose to make before the first release, from
the live check of 2026-11-20
([`2026-11-20-live-check.md`](../../design/2026-11-20-live-check.md)). Specs,
all amended on 2026-11-20 and Accepted: SPEC-0002, SPEC-0003, SPEC-0004,
SPEC-0006, SPEC-0007, SPEC-0008. No spec is deferred. Stage plan 0003 (M3
release) resumes at its T6 once this stage is done.

Exit: every amended requirement has a passing test that names it, and the
live check, repeated on a copy of a real notes root and a scratch root, shows
none of the findings marked for this stage.

## Slices

| Slice | Tasks | Ends with |
|---|---|---|
| A — Identifier search | T1–T2 | `search` for `IT-104` returns only notes holding `IT-104`, says when it stopped at the limit, and shows tags |
| B — Graph and refs | T3–T5 | `context` lists every edge between its notes, once each; refs say how they resolved; an unknown ref gets suggestions |
| C — Read and list output | T6–T7 | A section read lists only its own links; a missing section has its own code; `list` shows tags and a date for every note |
| D — Writes | T8–T10 | `append` starts a paragraph, `replace_section` keeps the blank line, writes list unresolved links, move and delete name the linkers |
| E — Git | T11–T13 | Commit messages name paths; editor changes get their own commits, also those made while memctl was stopped |
| F — Lock message | T14 | The refusal names the given root and appears once in a client's log |
| G — Close | T15–T16 | Benchmarks rerun, CHANGELOG, live check repeated, specs `Implemented` |

## File map

| File | Change | Why |
|---|---|---|
| `internal/index/search.go` | modify | Compound tokens, query terms, `Result{Hits, Total}`, tags in hits (SPEC-0002 R7, R24, R25) |
| `internal/index/resolve.go` | modify | `Step` and `Also` (SPEC-0002 R23) |
| `internal/index/graph.go` | modify | Edges among listed notes, duplicates once (SPEC-0002 R17) |
| `internal/index/index.go` | modify | `ModTime` in `Entry` (SPEC-0002 R26) |
| `internal/index/bench_test.go` | modify | Rerun with compound tokens (SPEC-0002 R22) |
| `internal/note/fresh.go` | modify | Hand external paths to the committer (SPEC-0003 R18) |
| `internal/note/service.go` | modify | `Committer` interface gains external changes and the wait of SPEC-0008 R16 |
| `internal/note/edit.go` | modify | `append` paragraph rule, `replace_section` spacing, `section_not_found` (SPEC-0007 R16, R17) |
| `internal/note/move.go`, `delete.go` | modify | Linker paths, up to 10 (SPEC-0007 R23, R24) |
| `internal/note/lock.go` | modify | Given and resolved root in `HeldError` (SPEC-0006 R5) |
| `internal/gitlog/committer.go` | modify | Path messages, move message, external queue, ordering, wait before a tool write, flush on shutdown (SPEC-0008 R5, R12, R14, R16) |
| `internal/gitlog/run.go` | modify | `status` and `config --get` allowed (SPEC-0008 R2) |
| `internal/gitlog/scan.go` | create | Start-up scan of uncommitted notes (SPEC-0008 R15) |
| `internal/mcpserver/search.go` | modify | Tags, total line, last line (SPEC-0004 R19, R29) |
| `internal/mcpserver/read.go` | modify | Resolution item, section links, `section_not_found` (SPEC-0004 R16, R27) |
| `internal/mcpserver/context.go` | modify | First line, resolution line; `list` lines with tags and dates (SPEC-0004 R22, R27, R30) |
| `internal/mcpserver/render.go` | modify | Shared not-found suggestions, linker lists, unresolved links (SPEC-0004 R28, SPEC-0007 R23, R24, R29) |
| `internal/mcpserver/write.go`, `edit.go`, `move.go`, `delete.go` | modify | Unresolved links, linker paths, `section_not_found`, count `unresolved` |
| `internal/mcpserver/result.go` | modify | Code `section_not_found` |
| `internal/mcpserver/tools.go` | modify | Descriptions of `search`, `context`, `list`, `note_read`, `note_edit`, `note_move`, `note_delete` |
| `cmd/memctl/mcp.go` | modify | Lock message once, plain line only to a terminal; start-up scan wiring (SPEC-0006 R5, SPEC-0008 R15) |
| `*_test.go`, `testdata/` | modify | Tests named `SPEC0002_R24` style; golden files updated |
| `CHANGELOG.md` | modify | Rewrite the `[Unreleased]` `Added` entries the stage changes (nothing has been released, so no `Changed` entries) |
| `docs/design/YYYY-MM-DD-live-check-fixes.md` | create (T16) | Verification record of the repeated live check |
| Specs and `docs/specs/README.md` | modify | Acceptance boxes; `Implemented` |

Interface change between `note` and `gitlog`: `Committer` gains
`External(paths []string)` and `WaitExternal(path string) error`;
`Change` gains `OldPath` for a move. `note` still does not import `gitlog`.

## Dependency order

```
T1 → T2                       (slice A)
T3 → T4 → T5                  (slice B; T4 needs T3's Resolution only for tests)
T6, T7                        (slice C; independent)
T8 → T9 → T10                 (slice D; T10 needs T5's render helper)
T11 → T12 → T13               (slice E)
T14                           (slice F; independent)
all → T15 → T16               (slice G)
```

Slices A, C, E and F are independent of each other; B before D.

## Tasks

Each task: write the tests named after the requirements, run them and see them
fail for the expected reason, commit them (`test(<pkg>): … (spec: SPEC-NNNN)`),
implement, `make verify`, commit (`feat(<pkg>): …` or `fix(<pkg>): …
(spec: SPEC-NNNN)`).

### Slice A — Identifier search

- [x] **T1 — Compound tokens.** Tokenizer emits parts and compounds; indexing
  uses both and counts both in field lengths; query terms are compounds plus
  parts outside compounds; `Result{Hits, Total}`; tags in hits. Update the
  hand-computed BM25F fixture values.
  *SPEC-0002 R7, R24, R25.*
- [x] **T2 — `search` output.** Hit line with tags; `showing the first <n>`
  first line and the closing hint when `Total` exceeds the hits; description
  says that an identifier matches whole. Golden files.
  *SPEC-0004 R19, R29.*

### Slice B — Graph and refs

- [x] **T3 — Edges among listed notes.** After the walk, collect every resolved
  edge whose ends are both listed (with `relTypes`), drop duplicates by
  source, type and target at the first line, keep the start's dangling and
  ambiguous edges, order as before.
  *SPEC-0002 R17.*
- [ ] **T4 — Resolution step.** `Step` and `Also` in `Resolve`; `note_read`
  metadata item and `context` line; `context` first line `plus the note
  itself`; description names the edge rule.
  *SPEC-0002 R23; SPEC-0004 R22, R27.*
- [ ] **T5 — Not-found suggestions.** A dangling ref runs `Search` with the
  ref's text, limit 3; two sentences of R28; nothing logged.
  *SPEC-0004 R28.*

### Slice C — Read and list output

- [ ] **T6 — Section reads.** `Links in this section:` with the links on the
  section's lines; `section_not_found` with the headings in `note_read`; new
  code in `result.go` and the code test.
  *SPEC-0004 R16; error table.*
- [ ] **T7 — List lines.** `ModTime` in entries; lines with tags and
  `updated` or `file changed`; description names both dates.
  *SPEC-0002 R26; SPEC-0004 R24, R30.*

### Slice D — Writes

- [ ] **T8 — `append` and `replace_section`.** Paragraph rule for `append`;
  one blank line after the heading for `replace_section`; `section_not_found`
  with the `append` advice; description of `append`. Golden files of R16 and
  R17.
  *SPEC-0007 R16, R17.*
- [ ] **T9 — Linker lists.** `note_move` and `note_delete` name up to 10
  linkers, sorted, with `and <k> more`.
  *SPEC-0007 R23, R24.*
- [ ] **T10 — Unresolved links.** After `note_write` and `note_edit`, list the
  note's dangling and ambiguous links in the `Links` format; count
  `unresolved` in the log line.
  *SPEC-0007 R29.*

### Slice E — Git

- [ ] **T11 — Path messages.** Queue entries carry paths and the old path of a
  move; one-entry, move and batch messages of R5.
  *SPEC-0008 R5.*
- [ ] **T12 — External changes.** `status` and `config --get` in the runner;
  external queue apart from tool entries; message forms; repository identity
  or `--git-author`; committed before a tool batch due at the same moment;
  freshness check hands `Added`, `Changed`, `Removed` paths; a tool write to a
  path with a pending external change waits for that commit; shutdown flushes
  both queues.
  *SPEC-0003 R18; SPEC-0008 R2, R12, R14, R16.*
- [ ] **T13 — Start-up scan.** `git status --porcelain=v1 -z
  --untracked-files=all` after the repository test; queue the note paths and
  deleted `.md` paths as external changes.
  *SPEC-0008 R15.*

### Slice F — Lock message

- [ ] **T14 — Lock refusal.** `HeldError` carries the given and the resolved
  root; the `ERROR` line holds the message; the plain line only when stderr is
  a character device.
  *SPEC-0006 R5.*

### Slice G — Close

- [ ] **T15 — Benchmarks and changelog.** Rerun `BenchmarkBuild`,
  `BenchmarkSearch`, `BenchmarkWalk` and record them below; rewrite the
  `[Unreleased]` entries the stage changed; README where tool output or git
  behaviour is described.
  *SPEC-0002 R22.*
- [ ] **T16 — Live check and close.** `make build`; the maintainer repeats the
  live check of 2026-11-20 with the same scripted prompt, extended by an
  editor change between two tool writes and a note edited while memctl is
  stopped. Write the verification record under `docs/design/` and list it in
  its README; tick every acceptance box; set the six specs to `Implemented`
  with their index rows; tick this plan. Commit
  `docs: live-check fixes implemented (spec: SPEC-0002..0004, SPEC-0006..0008)`.

## Requirement coverage

| Spec | Requirements | Tasks |
|---|---|---|
| SPEC-0002 | R7, R24, R25 | T1 |
| SPEC-0002 | R17 | T3 |
| SPEC-0002 | R23 | T4 |
| SPEC-0002 | R26 | T7 |
| SPEC-0002 | R22 | T15 |
| SPEC-0003 | R18 | T12 |
| SPEC-0004 | R19, R29 | T2 |
| SPEC-0004 | R22, R27 | T4 |
| SPEC-0004 | R28 | T5 |
| SPEC-0004 | R16, `section_not_found` | T6 |
| SPEC-0004 | R24, R30 | T7 |
| SPEC-0006 | R5 | T14 |
| SPEC-0007 | R16, R17 | T8 |
| SPEC-0007 | R23, R24 | T9 |
| SPEC-0007 | R29 | T10 |
| SPEC-0008 | R5 | T11 |
| SPEC-0008 | R2, R12, R14, R16 | T12 |
| SPEC-0008 | R15 | T13 |

## Definition of done

- `make verify` passes.
- Every amended or added requirement has at least one passing test whose name
  carries it; every acceptance box of the six specs is ticked; the specs and
  their index rows say `Implemented`.
- The repeated live check is recorded under `docs/design/` and listed in its
  README, and shows none of the findings of this stage.
- `CHANGELOG.md` `[Unreleased]` describes the behaviour as it now is.

## Verification notes

(Benchmarks of T15.)
