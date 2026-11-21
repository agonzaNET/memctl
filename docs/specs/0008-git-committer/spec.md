# SPEC-0008 - Git committer

## Status

Accepted

> Amended on 2026-11-20 for findings L10 and L11 of the live check
> ([`2026-11-20-live-check.md`](../../design/2026-11-20-live-check.md)),
> delivered by stage plan 0004: R2 and R5 changed; R14 to R16 added; the scope
> now covers changes made outside memctl. R2 amended again on 2026-11-20 after
> the release audit: every git call reads paths literally, never as patterns
> (stage plan 0004 T17). R17 added on 2026-11-20 after the repeated live
> check: a `WARN` at start names a missing git identity key (stage plan 0004
> T20).
>
> Amended on 2026-11-21 for the shared instance
> ([SPEC-0009](../0009-shared-instance/spec.md), ADR-0021), for v0.3.0: R12
> changed: the shutdown starts when the last session ends, not on stdin EOF
> alone.

<!--
Status values: Draft | Accepted | Implemented | Superseded by SPEC-NNNN | Deprecated
Lifecycle: Draft (under review) → Accepted (approved; the code may be built,
but at least one acceptance criterion, for example a live test, is still
open) → Implemented (shipped; code satisfies this spec and all acceptance
criteria are ticked). Superseded by SPEC-NNNN and Deprecated are terminal.
-->

**Governing ADRs:** ADR-0002, ADR-0005, ADR-0014, ADR-0016, ADR-0021

## Purpose

Record every tool write as a git commit in the notes root, so each change can
be reviewed and undone, and AI edits stay separable from human edits (design
D5, D10, D13). Commit in the background, batch bursts of writes, never touch
anything a tool did not change, and never push.

## Scope

**In scope**

- `internal/gitlog`: the queue fed by the write tools (SPEC-0007 R5), the quiet
  window, the commit through the `git` CLI, the final flush on shutdown, and the
  repository test shared with `memctl check` (SPEC-0005 R7).
- `cmd/memctl`: the `--git-author` flag, and the shutdown rules of ADR-0014 that
  SPEC-0004 R6 left to this spec.

**Out of scope**

- `git push`, `pull`, `fetch`, branches, merges, tags: never.
- Changes outside the notes, and changes to notes that neither the freshness
  check nor the start-up scan of R15 finds.

## Inputs

| Flag | Environment variable | Default | Rule |
|---|---|---|---|
| `--git-author` | `MEMCTL_GIT_AUTHOR` | `memctl <memctl@localhost>` | `Name <email>`, name 1–100 characters, email with one `@`, no line break |

- From each successful write: the changed paths (relative to the root), the
  tool name and the note title; for a move, the old and the new path.
- From each freshness check (SPEC-0003 R18) and from the start-up scan (R15):
  the paths of notes changed outside memctl.

## Outputs

- Commits in the repository that contains the root, on its current branch.
- Log lines: `INFO` `committed` with `paths` (count) and `commit` (short hash);
  `WARN` at start when the root is not in a git work tree or `git` is missing;
  `WARN` `incomplete git identity` at start when the repository lacks
  `user.name` or `user.email` (R17);
  `ERROR` `commit failed` with the git exit status and the failure class.

## Behaviour & Requirements

### Repository

- **R1** — At start, `memctl mcp` MUST test the root with
  `git -C <root> rev-parse --is-inside-work-tree` (5 s timeout). When the answer
  is not `true`, or `git` cannot be run, it MUST log one `WARN` line, and the
  committer MUST accept and drop every batch without running git.
- **R2** — The committer MUST only run `git` with these subcommands:
  `rev-parse`, `check-ignore`, `add`, `commit`, and for external changes
  `status` and `config --get`. Every call MUST use `-C <root>`,
  a 30 s timeout, `GIT_TERMINAL_PROMPT=0`, and `-c commit.gpgSign=false`.
  Every call but `check-ignore` MUST use `GIT_LITERAL_PATHSPECS=1`, so a path
  is never read as a pattern and `[draft] x.md` or `*.md` names only that
  file; `check-ignore` reads its paths literally and refuses that setting.

### Queue and timing

- **R3** — Each write MUST add its paths and a `(tool, title, path)` entry to the
  queue without blocking the tool call.
- **R4** — A batch MUST be committed when 2 s pass with no new entry, or 30 s
  after the first entry of the batch, whichever comes first.
- **R5** — One batch MUST be one commit. A batch of one entry MUST have the
  message `memory: <tool> <path>`, and a move `memory: note_move <old path> → <new path>`.
  A batch of several MUST have the subject `memory: <n> changes` and a body
  with one line per entry, `- <tool> <path>` (a move
  `- note_move <old path> → <new path>`), in queue order.

### The commit

- **R6** — For a batch, the committer MUST drop paths that git ignores
  (`check-ignore`), stage the rest with `git add -A -- <paths>` (deletions and
  moves included), and commit with `git commit --only -- <paths>`, so content
  that a person staged for other paths is neither committed nor unstaged.
- **R7** — Author and committer MUST both be the `--git-author` identity (set
  through `GIT_AUTHOR_NAME`, `GIT_AUTHOR_EMAIL`, `GIT_COMMITTER_NAME`,
  `GIT_COMMITTER_EMAIL`), so the commit works without a git identity in the
  user's config.
- **R8** — Git hooks MUST run as configured. A hook that fails makes the commit
  fail (R10).
- **R9** — When git reports nothing to commit (the files equal `HEAD`), the batch
  MUST succeed with no commit and no `ERROR`.
- **R10** — When `add` or `commit` fails because `.git/index.lock` exists, the
  committer MUST retry up to 3 times, 1 s apart. Any other failure, or the last
  retry failing, MUST log one `ERROR` `commit failed` line and drop the batch;
  the files stay on disk and show in `git status`. A failure MUST NOT affect
  tool calls.
- **R11** — Log lines MUST NOT contain titles, paths, commit messages or git
  output text; the failure class is one of `index_locked`, `hook_failed`,
  `timeout`, `git_error`.

### Changes made outside memctl (design, "Git")

- **R14** — External changes MUST be queued apart from tool entries and
  committed in their own commits, never in a tool's commit: one path gives the
  message `memory: external change <path>`, several give the subject
  `memory: <n> external changes` and a body with one line `- <path>` each,
  sorted. Author and committer MUST be the repository's identity when
  `git config --get user.name` and `git config --get user.email` in the root
  both return a value, and the `--git-author` identity otherwise. They MUST
  follow the timing of R4 and the rules of R6, R8 to R11, and be committed
  before a tool batch that is due at the same moment.
- **R15** — At start, after R1, the committer MUST run
  `git status --porcelain=v1 -z --untracked-files=all` in the root and queue as
  external changes the paths that the walk treats as notes (SPEC-0003 R3 to R5)
  and deleted paths ending in `.md`, so changes made while memctl was not
  running are committed too.
- **R16** — Before a tool writes a path that has a pending external change, the
  write MUST wait until the pending external changes are committed (at most
  the 30 s timeout of R2); when that commit fails, the write MUST go on after
  the `ERROR` line of R10. A person's edit made within the quiet window after a
  tool wrote the same path, and before that tool's commit, is a known limit:
  the two changes end up in one commit.
- **R17** — At start, after R1 and only when the root is in a git work tree,
  when `git config --get user.name` or `git config --get user.email` in the
  root gives no value, `memctl mcp` MUST log one `WARN` line
  `incomplete git identity` with `missing` (the keys without a value, comma
  separated, `user.name` first) and `author` (the `--git-author` identity that
  R14 uses instead for external changes). With both keys set it MUST log
  nothing.

### Shutdown (ADR-0014)

- **R12** — On the shutdown context (the end of the last session, SIGINT or
  SIGTERM; SPEC-0004 R6), after the tool
  call in progress finishes, the committer MUST commit the pending external
  changes and the pending batch at once,
  without waiting for the quiet window, and `memctl mcp` MUST exit only after
  that commit returns (success or failure).
- **R13** — A second SIGINT or SIGTERM during shutdown MUST exit at once with
  status 1, without waiting for the commit, after one `WARN` line.

## Error handling

| Failure | Result |
|---|---|
| Root not in a git work tree, or `git` missing | `WARN` at start; writes work, nothing is committed (R1) |
| Repository without `user.name` or `user.email` | `WARN` at start naming the key; external changes use `--git-author` (R14, R17) |
| `index.lock` present | Up to 3 retries, then `ERROR` (R10) |
| Hook fails, timeout, other git error | `ERROR` with the class; files stay uncommitted (R10) |
| Bad `--git-author` | Usage error, status 2 (SPEC-0004 R1) |

## Acceptance criteria — "Done looks like"

- [x] **R1** — A root outside git, a subfolder of a repository, and a `PATH`
  without `git`.
- [x] **R2** — A fake `git` on `PATH` records only the allowed subcommands and
  options, `status` and `config --get` included.
- [x] **R2 (literal paths)** — Tool writes to `[draft] x.md` and `*.md` commit
  only those files, and a person's unstaged change to `d x.md` stays unstaged.
- [x] **R3, R4** — With a fake clock: one write commits after 2 s; writes every
  second commit at 30 s; tool calls return before the commit.
- [x] **R5** — Messages for one entry, a move, and three entries with a move
  among them; paths, not titles.
- [x] **R6** — With a person's staged change in another file, a tool commit
  contains only the tool's paths, and the person's change stays staged; a move
  and a delete are committed as such; an ignored path is dropped.
- [x] **R7** — In a repository with no `user.name`, the commit succeeds with
  the default author and committer; `--git-author "Ann <ann@example.org>"` is
  used when given.
- [x] **R8** — A failing `pre-commit` hook gives one `ERROR` with `hook_failed`.
- [x] **R9** — An edit reverted before the batch commits gives no commit.
- [x] **R10, R11** — A held `index.lock` released after 2 s: commit after a retry;
  held for 10 s: `ERROR` `index_locked`; log lines hold no title or path.
- [x] **R12** — An end-to-end test writes a note and closes stdin at once; the
  commit exists when the process has exited.
- [x] **R12 (again)** — An editor change found by a check, then stdin closed at
  once: the external commit exists when the process has exited.
- [ ] **R12 (shared)** — A note written through a forwarded session, then the
  owner's and the forwarder's stdin closed at once: the commit exists, with
  the owner's author, when the owner has exited.
- [x] **R14** — An editor change to a note, then a tool write to another note:
  two commits, the external one first, with the repository identity when
  `user.name` and `user.email` are set and `--git-author` when not; three
  external paths give one commit with the sorted body.
- [x] **R15** — A note edited, one added and one deleted while memctl was not
  running are committed as external changes after start; an untracked
  `x.txt` and a file under an ignored folder are not.
- [x] **R17** — A repository with `user.name` only gives one `WARN` with
  `missing` `user.email` and the author; with neither, `user.name, user.email`;
  with both, no line; a root outside git gives only the `WARN` of R1.
- [x] **R16** — An editor change to a note, then `note_edit` on it within the
  quiet window: the external commit holds the editor change only, the tool
  commit holds the edit only; with the external commit failing, the edit is
  written and committed after the `ERROR` line.
- [x] **Live check (again)** — The exit check of stage plan 0004 repeats the git
  checks of the live check of 2026-11-20, with an editor change between two
  tool writes.
- [x] **R13** — A second SIGTERM during a slow fake commit exits at once with
  status 1.

## References

- Design: D5, D10, D12, D13, D21, "Git" in
  [`../../design/solution-design.md`](../../design/solution-design.md).
- [ADR-0014](../../adr/0014-shutdown-flushes-pending-commits.md).
- SPEC-0004 (`memctl mcp`), SPEC-0005 (`memctl check`), SPEC-0007 (write tools).
- SPEC-0003 R18 (external changes found by the freshness check).
- Live check of 2026-11-20, findings L10, L11:
  [`2026-11-20-live-check.md`](../../design/2026-11-20-live-check.md).
