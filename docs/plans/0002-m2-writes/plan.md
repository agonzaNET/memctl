# Stage plan 0002 — M2 writes

Stage: design milestone M2. Specs: SPEC-0006 (instance lock), SPEC-0007 (write
tools and atomic writes), SPEC-0008 (git committer). All Accepted. No earlier
spec plan exists for these specs.

Exit of M2 (design): a Claude Code session creates, edits and renames notes,
every change is one git commit, and a concurrent edit returns a conflict
instead of a silent overwrite.

## Slices

The tasks are ordered as four slices, each ending in something that runs:

| Slice | Tasks | Ends with |
|---|---|---|
| A — First committed write | T1–T5 | Claude Code creates a note with `note_write`; one git commit appears, also when the client closes at once |
| B — Edits | T6–T7 | `note_edit` with all three ops; a stale `expected_sha` gives `conflict` |
| C — Move and delete | T8–T9 | `note_move` with aliases and `note_delete` with linker counts |
| D — Completeness | T10–T13 | Git failure handling, final tool text, robustness, live check |

No spec is deferred.

## File map

| File | Change | Why |
|---|---|---|
| `internal/note/lock.go` | create | Lock file, `.memory/` and its `.gitignore`, `flock` (SPEC-0006) |
| `internal/note/write.go` | create | Atomic write, re-hash before rename, folder creation, index update, hand-off to the committer (SPEC-0007 R1–R7, R28) |
| `internal/note/filename.go` | create | File name from a title (SPEC-0007 R10) |
| `internal/note/ops.go` | create | `Write`, `Edit`, `Move`, `Delete` operations of the service (SPEC-0007 R8–R24) |
| `internal/note/service.go` | modify | `Options` with clock, committer and fault hooks; `Open` takes them |
| `internal/parse/fmedit.go` | create | Line-preserving frontmatter editor: set a scalar, set a flow list, insert a key (SPEC-0007 R8). In `parse` because it owns yaml.v3 (ADR-0010) and is needed to quote values only when YAML needs it |
| `internal/index/graph.go` | modify | `Linkers(path, names)`: count of notes linking to a path or name (SPEC-0007 R23, R24) |
| `internal/gitlog/committer.go` | create | Queue, quiet window, max delay, batch messages, flush on shutdown (SPEC-0008 R3–R5, R12) |
| `internal/gitlog/run.go` | create | The single `git` runner: allowed subcommands, `-C`, timeouts, environment, failure classes, `index.lock` retry (SPEC-0008 R2, R6–R11) |
| `internal/gitlog/repo.go` | modify | Use the runner, so `IsRepo` gets `-c commit.gpgSign=false` too (SPEC-0008 R2); keeps its 5 s timeout (R1) |
| `internal/mcpserver/write.go`, `edit.go`, `move.go`, `delete.go` | create | The four tools: handlers, rendering, notices |
| `internal/mcpserver/tools.go` | modify | Descriptions, schemas, annotations of the four tools between the agent-facing markers |
| `internal/mcpserver/result.go` | modify | Codes `exists`, `conflict`, `no_match`, `not_unique` |
| `internal/mcpserver/log.go` | modify | Counts `bytes`, `paths`, `linkers` |
| `cmd/memctl/config.go` | modify | `--git-author` / `MEMCTL_GIT_AUTHOR` and its validation |
| `cmd/memctl/mcp.go`, `lifecycle.go` | modify | Start order: root → lock → load → repo test → committer; shutdown: finish call, flush, exit; second signal exits at once |
| `*_test.go`, `testdata/` | create | Tests named `SPEC0006_R3` style; golden files for frontmatter edits and tool results; fake `git` script; fake clock |
| `CHANGELOG.md` | modify | `Added` entries in T1, T5, T7, T9 |
| `docs/design/YYYY-MM-DD-m2-live-check.md` | create (T13) | Verification record of the live check |
| `docs/specs/0006-…`, `0007-…`, `0008-…/spec.md`, `docs/specs/README.md` | modify | Acceptance boxes; status `Implemented` once each spec is complete |

Interface between `note` and `gitlog`: `note` defines
`type Committer interface { Enqueue(Change) }` with
`Change{Tool, Title string; Paths []string}`; `gitlog.Committer` implements
it, and `cmd/memctl` wires the two. `note` does not import `gitlog`. Tests of
`note` use a recording stub.

## Dependency order

```
T1 → T2 → T3 → T4 → T5        (slice A)
T5 → T6 → T7                  (slice B)
T7 → T8 → T9                  (slice C)
T9 → T10, T11 → T12 → T13     (slice D)
```

T10 and T11 are independent of each other. T6 only needs T2 and could start
earlier; it stays in slice B to keep slice A thin.

## Tasks

Each task: write the tests named after the requirements, run them and see them
fail for the expected reason, commit them
(`test(<pkg>): … (spec: SPEC-NNNN)`), implement, `make verify`, commit
(`feat(<pkg>): … (spec: SPEC-NNNN)`). A task that touches two specs names both
in the commit.

### Slice A — First committed write

- [x] **T1 — Instance lock.** `.memory/` (0700) and `.memory/.gitignore`
  (`*`, 0600, never changed when present); non-blocking `flock`, `O_CREATE`,
  0600, close-on-exec; PID written after the lock; refusal message with or
  without PID, status 1; create/open failures name path and operation; lock
  file never deleted. Wire into `memctl mcp` after root resolution, before the
  load. Tests: two processes, empty lock file, read-only root, `SIGKILL` then
  immediate retake, `git status` clean. CHANGELOG: one `memctl mcp` per root.
  *SPEC-0006 R1–R8.*
- [x] **T2 — Atomic write core.** Service `Options` (clock, committer,
  fault hooks before rename and before the re-hash). Dot-prefixed temporary
  file in the target folder, `fsync`, mode of the replaced file or 0644,
  `rename`, folder `fsync`; temporary file removed on any failure. Re-hash just
  before rename → `conflict`. Missing folders 0755; empty folders kept. Index
  `Put`/`Remove` after the write. Hand-off of paths, tool and title; nothing on
  failure. Target rejected when the walk would skip it (dot-segment,
  `.gitignore`). Local-time timestamps from the clock.
  *SPEC-0007 R1–R7, R28.*
- [x] **T3 — `note_write`.** File name rule; `exists` for the path and for a
  case-only clash; body starting with a frontmatter delimiter rejected;
  frontmatter `title`, `type`, `tags`, `created`, `updated` in order, blank
  line, body, one final newline; shared-title notice. Tool with schema,
  annotations, a first description, result lines, new codes `exists` and
  `conflict` in `result.go`, log counts `bytes` and `paths`. Committer is a
  logging stub in `cmd/memctl` until T4.
  *SPEC-0007 R10–R14; R25–R27 for `note_write`.*
- [x] **T4 — Git committer.** The `git` runner (allowed subcommands, `-C`,
  30 s timeout, `GIT_TERMINAL_PROMPT=0`, `-c commit.gpgSign=false`); `IsRepo`
  through it. Repository test at start: `WARN` and drop-all mode outside a
  work tree or without `git`. Non-blocking queue; 2 s quiet window and 30 s
  maximum with an injectable clock; one batch = one commit; message forms
  with line breaks in titles replaced. `check-ignore` filter, `add -A --`,
  `commit --only --`; author and committer from `--git-author` (flag,
  environment, default, validation as a usage error); nothing-to-commit is
  success. `committed` log line with counts only. Fake `git` on `PATH`
  records the calls.
  *SPEC-0008 R1–R7, R9, R11 (for the lines added here).*
- [x] **T5 — Shutdown flush and first end-to-end write.** *(The manual
  Claude Code check moves to the T13 live check, which the maintainer runs.)* Shutdown context
  passed to the committer; the call in progress finishes, the pending batch is
  committed at once, `memctl mcp` exits after the commit returns; a second
  signal logs one `WARN` and exits 1 at once. End-to-end test: `note_write`
  over pipes, stdin closed at once, commit present after exit. Manual check:
  Claude Code creates a note in a scratch git root. CHANGELOG: `note_write`,
  git auto-commit, `--git-author`.
  *SPEC-0008 R12, R13; SPEC-0004 R6 (shutdown completed).*

### Slice B — Edits

- [x] **T6 — Frontmatter editor.** `parse.EditFrontmatter`: replace a key's
  line or block list in place, insert a new key before the closing delimiter,
  keep every other byte; flow-style lists, quoting only when yaml.v3 would
  misread the plain value. Golden tests with comments, unknown keys, block and
  flow lists, quoted values; no frontmatter and `bad_frontmatter` refused.
  *SPEC-0007 R8, R9.*
- [x] **T7 — `note_edit`.** `expected_sha` check after the freshness check →
  `conflict` with current sha and fenced content; `append`,
  `replace_section` (heading lookup as `note_read`, `not_found` with the
  headings), `find_replace` (`no_match`, `not_unique` with count, frontmatter
  included); `updated` set unless R9; byte-identical result → no write, no
  hand-off, `is unchanged` line. Tool, codes `no_match` and `not_unique`.
  CHANGELOG: `note_edit`.
  *SPEC-0007 R15–R19; R25–R27 for `note_edit`.*

### Slice C — Move and delete

- [ ] **T8 — `note_move`.** Argument rules; new path from `new_title` (R10);
  `exists`; move onto itself writes nothing; `rename`, then atomic rewrite with
  `title`, `aliases` (old path without `.md`, old title, no duplicates) and
  `updated`; `new_title` on a note without usable frontmatter →
  `invalid_argument`. `index.Linkers` for the result's count and the notice
  about aliases and `memctl check`. Both paths handed to the committer, so git
  records a rename. No other note changes.
  *SPEC-0007 R9 (move part), R20–R23; R25–R27 for `note_move`.*
- [ ] **T9 — `note_delete`.** Remove file and index entry; dangling-linker
  count; git-history notice; second delete `not_found`. CHANGELOG:
  `note_move`, `note_delete`.
  *SPEC-0007 R24; R25–R27 for `note_delete`.*

### Slice D — Completeness

- [ ] **T10 — Git failures.** `index.lock` retry (3 × 1 s); classes
  `index_locked`, `hook_failed`, `timeout`, `git_error`; one `ERROR`
  `commit failed`, batch dropped, tool calls unaffected; hooks run as
  configured. Tests: a person's staged change in another file stays staged and
  out of the tool commit; move and delete committed as such; ignored path
  dropped; lock released after 2 s vs held 10 s; failing `pre-commit`; no title
  or path in any log line.
  *SPEC-0008 R6, R8, R10, R11.*
- [ ] **T11 — Tool text and log lines.** Final descriptions of the four tools
  under ADR-0013 (side effect, commit within seconds, never pushed, sha from
  `note_read`, every error code); annotation table test; `check-docs.sh`
  check 5 passes; captured log lines carry `bytes`, `paths`, `linkers` and no
  argument value.
  *SPEC-0007 R25–R27.*
- [ ] **T12 — Robustness.** Fault injection before `rename` and before the
  re-hash (R1–R3); property test: random sequences of the four writes and
  editor-style file changes, after each step the index equals a fresh load and
  the stub received the expected paths (R4, R5); `-race` with concurrent tool
  calls and a running committer; end-to-end test of a burst of writes giving
  one commit.
  *SPEC-0007 R1–R5; SPEC-0008 R3, R4.*
- [ ] **T13 — Live check and close.** Build with `make build`; in a Claude
  Code session on a scratch git root: create, edit, rename and delete notes;
  edit one in an editor between a read and an edit and see `conflict`; check
  `git log` (one commit per change or batch, author `memctl`); start a second
  client on the same root and see the refusal. Write the verification record
  under `docs/design/` and list it in its README. Tick every acceptance box of
  SPEC-0006 to SPEC-0008, set each to `Implemented` with its index row; tick
  this plan. Commit `docs: M2 implemented (spec: SPEC-0006..0008)`.
  The open M1 live check (SPEC-0004) can be run in the same session.

## Requirement coverage

| Spec | Requirements | Tasks |
|---|---|---|
| SPEC-0006 | R1–R8 | T1 |
| SPEC-0007 | R1–R7, R28 | T2 (R1–R5 again in T12) |
| | R8, R9 | T6 (R9 move part in T8) |
| | R10–R14 | T3 |
| | R15–R19 | T7 |
| | R20–R23 | T8 |
| | R24 | T9 |
| | R25–R27 | T3, T7, T8, T9 per tool; T11 final |
| SPEC-0008 | R1–R5, R7, R9 | T4 (R3, R4 again in T12) |
| | R6 | T4 (stage and commit), T10 (staged-content and ignore tests) |
| | R8, R10 | T10 |
| | R11 | T4, T10 |
| | R12, R13 | T5 |

## Definition of done

- `make verify` passes: `go test ./...` (golden files, end-to-end tests, fake
  `git`, import scan), `go vet ./...`, `gofmt -l .` empty, `check-docs.sh`
  including the agent-facing text check of `tools.go`.
- `go test -race ./...` passes.
- Tests that need a real `git` skip with a message when `git` is missing; the
  development VM has `git`, so they run there.
- Every requirement of SPEC-0006 to SPEC-0008 has at least one passing test
  whose name carries it (`SPEC0007_R15` style); every acceptance box is ticked;
  the three specs and their index rows say `Implemented`.
- The live check is recorded under `docs/design/` and listed in its README.
- `CHANGELOG.md` `[Unreleased]` lists the instance lock, the four write tools,
  git auto-commit and `--git-author`.

## Notes

- The dev VM has Go 1.26.8 in `~/sdk/go` (not on `PATH`); commands run with
  `PATH=$HOME/sdk/go/bin:$PATH`.
- `flock` is Linux and macOS only, matching the build targets of ADR-0012.
- Commits are local on `main`; the maintainer pushes.
