# SPEC-0003 - Notes root, startup load and freshness check

## Status

Accepted

<!--
Status values: Draft | Accepted | Implemented | Superseded by SPEC-NNNN | Deprecated
Lifecycle: Draft (under review) → Accepted (approved; the code may be built,
but at least one acceptance criterion, for example a live test, is still
open) → Implemented (shipped; code satisfies this spec and all acceptance
criteria are ticked). Superseded by SPEC-NNNN and Deprecated are terminal.
-->

**Governing ADRs:** ADR-0002, ADR-0005, ADR-0016, ADR-0017

## Purpose

Decide which files under the notes root are notes, load them all into the
index at start, and keep the index equal to the disk before every tool call,
so that edits made in any editor are seen on the next call (design D14, D15).
Also turn a note path given by a tool into a safe file path inside the root.

## Scope

**In scope**

- `internal/note`: opening the root, the walk that finds notes, the parallel
  startup load, the freshness check, the path safety rule, and the one mutex
  that serialises every call into the note service and the index.

**Out of scope**

- Parsing (SPEC-0001) and indexing (SPEC-0002).
- The tools that call the service (SPEC-0004), `memctl check` (SPEC-0005).
- Writes, the instance lock and git (M2 specs). This spec only provides the
  path rule they will use.
- A file watcher (design D15: none).

## Inputs

- The notes root from `--root`: a path to an existing directory, absolute or
  relative to the working directory, `~` already expanded by the shell.
- The files and directories under it.

## Outputs

- A loaded index (SPEC-0002) whose entries equal the notes on disk.
- `Stats{Added, Changed, Removed, Unchanged, Skipped}` per load and per check.
- Log lines (stderr, JSON): one `INFO` `notes loaded` line at start with the
  note count, skipped count and duration; one `DEBUG` line per freshness check
  with the counts and duration; `WARN` lines for skipped files. Log lines name
  no note path or content (ADR-0016), except that a skipped file's reason
  is logged with its path at `DEBUG` only.

## Behaviour & Requirements

### The root

- **R1** — `note.Open(root)` MUST resolve the root to an absolute path with
  symlinks evaluated once, and MUST fail with an error that names the path when
  it does not exist, is not a directory or cannot be read.
- **R2** — All paths the service hands to the index and the tools MUST be
  relative to the resolved root, slash-separated, without a leading `./`.

### Which files are notes

- **R3** — A note MUST be a regular file whose name ends in `.md` (lower case)
  and does not start with `.`.
- **R4** — The walk MUST skip every directory whose name starts with `.`
  (including `.git`, `.memory`, `.obsidian`, `.trash`), and MUST NOT follow
  symbolic links to files or directories.
- **R5** — The walk MUST skip files and directories matched by `.gitignore`
  files in the root and its subdirectories, with the pattern rules of git:
  blank lines and `#` comments, `!` negation, a trailing `/` for directories
  only, a leading or inner `/` anchoring to the `.gitignore`'s directory, `*`,
  `?`, `[…]` and `**`. A pattern in a deeper `.gitignore` MUST take precedence
  over a shallower one. `.git/info/exclude` and global excludes MUST NOT be read.
- **R6** — A note larger than 4 MiB MUST be skipped and counted in `Skipped`.
- **R7** — A note that cannot be read MUST be skipped and counted in `Skipped`.
  A skipped note's entry in the index MUST be removed. The startup load, and a
  check that skips a note it did not skip in the previous check, MUST log one
  `WARN` line with the count, per reason (`too_large`, `unreadable`); a note
  that stays skipped MUST NOT be reported again on every tool call.

### Startup load

- **R8** — `Open` MUST walk the root, read and parse every note in parallel
  with at most `runtime.GOMAXPROCS(0)` workers, compute each file's SHA-256, and
  add each to the index with `Meta{ModTime, Size, SHA256}`. The result MUST equal
  a sequential load (SPEC-0002 R2).
- **R9** — `Open` MUST log the `notes loaded` line (Outputs) and return the
  service only after the load is complete.

### Freshness check (design D15)

- **R10** — Before each tool call, the service MUST walk the root and stat every
  note. A note whose size or modification time differs from its index entry
  MUST be read and hashed; when the hash differs, it MUST be parsed and put
  again (`Changed`); when the hash is the same, only its meta MUST be updated
  (`Unchanged`). A note not in the index MUST be added (`Added`); an entry whose
  file is gone or no longer a note MUST be removed (`Removed`).
- **R11** — A change to a `.gitignore` file MUST take effect on the next check:
  notes it now ignores are removed and notes it no longer ignores are added.
- **R12** — The check MUST run inside the service mutex, so no tool call sees a
  half-updated index; tool calls MUST be serialised by the same mutex.
- **R13** — After a check, the index MUST equal what a fresh `Open` of the same
  root would load.

### Path safety

- **R14** — `note.SafePath(rel)` MUST accept a relative, slash-separated path
  ending in `.md` and return the absolute path inside the root. It MUST reject,
  with `note.ErrBadPath`: an empty path, an absolute path, a `\`, a NUL byte, a
  `.` or `..` segment, an empty segment, a segment that starts with `.`, and a
  path whose existing directories include a symbolic link.
- **R15** — `SafePath` MUST NOT create or modify anything.

### Performance

- **R16** — With 5,000 notes and no change on disk, a benchmark SHOULD show a
  freshness check under 30 ms on the development machine, and a startup load
  under 2 s. The numbers are recorded in the stage plan; missing one is a finding
  for the design's scale risk, not a failed test.

## Error handling

| Failure | Result |
|---|---|
| Root missing, not a directory, unreadable | `Open` returns an error naming the path; `memctl` exits non-zero (SPEC-0004) |
| A note unreadable or over 4 MiB | Skipped, counted, `WARN` line; the load or check goes on (R6, R7) |
| A subdirectory unreadable | Skipped like a file; its notes are removed from the index |
| A bad `.gitignore` line | Ignored, as git does |
| A bad tool path | `note.ErrBadPath`; the tool layer maps it to an invalid-argument error (SPEC-0004) |

## Acceptance criteria — "Done looks like"

- [ ] **R1, R2** — Open succeeds on a relative root and on a root reached through
  a symlink; fails on a missing path, a file and an unreadable directory.
- [ ] **R3, R4** — A fixture with `a.md`, `A.MD`, `.hidden.md`, `x.txt`,
  `.git/x.md`, `.obsidian/y.md`, `.trash/z.md`, a symlinked file and a
  symlinked folder loads only `a.md`.
- [ ] **R5** — Table test of `.gitignore` patterns against git's own answers
  (`git check-ignore`, run in the test when `git` is present): comments,
  negation, directory-only, anchored, `**`, character classes, nested override.
- [ ] **R6, R7** — A 5 MiB note and an unreadable note are skipped and counted;
  a note that becomes unreadable is removed on the next check with one `WARN`,
  and the check after it gives none.
- [ ] **R8, R9** — The parallel load equals a sequential load on a 500-note
  fixture (`-race` clean); the `notes loaded` line has the counts.
- [ ] **R10** — Edit, touch without change, add, delete and rename a note
  between two checks; the stats and the index match each case.
- [ ] **R11** — Adding `drafts/` to `.gitignore` removes `drafts/*.md` on the
  next check; removing the line adds them back.
- [ ] **R12** — Concurrent tool calls under `-race` never observe a partial
  check.
- [ ] **R13** — A property test of random file operations: after each check,
  the index equals a fresh `Open`.
- [ ] **R14, R15** — Table test of accepted and rejected paths, including a
  symlinked folder inside the root; the root is unchanged afterwards.
- [ ] **R16** — `BenchmarkOpen` and `BenchmarkCheck` exist; results recorded in
  the stage plan.

## References

- Design: D14, D15, "Change detection, concurrency and git" in
  [`../../design/solution-design.md`](../../design/solution-design.md).
- SPEC-0001, SPEC-0002; SPEC-0004 (MCP read tools), SPEC-0005 (`memctl check`) —
  planned.
