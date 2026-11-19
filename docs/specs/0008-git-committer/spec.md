# SPEC-0008 - Git committer

## Status

Accepted

<!--
Status values: Draft | Accepted | Implemented | Superseded by SPEC-NNNN | Deprecated
Lifecycle: Draft (under review) → Accepted (approved; the code may be built,
but at least one acceptance criterion, for example a live test, is still
open) → Implemented (shipped; code satisfies this spec and all acceptance
criteria are ticked). Superseded by SPEC-NNNN and Deprecated are terminal.
-->

**Governing ADRs:** ADR-0002, ADR-0005, ADR-0014, ADR-0016

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
- Committing files a tool did not write; uncommitted human edits stay as they
  are.

## Inputs

| Flag | Environment variable | Default | Rule |
|---|---|---|---|
| `--git-author` | `MEMCTL_GIT_AUTHOR` | `memctl <memctl@localhost>` | `Name <email>`, name 1–100 characters, email with one `@`, no line break |

- From each successful write: the changed paths (relative to the root), the
  tool name and the note title.

## Outputs

- Commits in the repository that contains the root, on its current branch.
- Log lines: `INFO` `committed` with `paths` (count) and `commit` (short hash);
  `WARN` at start when the root is not in a git work tree or `git` is missing;
  `ERROR` `commit failed` with the git exit status and the failure class.

## Behaviour & Requirements

### Repository

- **R1** — At start, `memctl mcp` MUST test the root with
  `git -C <root> rev-parse --is-inside-work-tree` (5 s timeout). When the answer
  is not `true`, or `git` cannot be run, it MUST log one `WARN` line, and the
  committer MUST accept and drop every batch without running git.
- **R2** — The committer MUST only run `git` with these subcommands:
  `rev-parse`, `check-ignore`, `add`, `commit`. Every call MUST use `-C <root>`,
  a 30 s timeout, `GIT_TERMINAL_PROMPT=0`, and `-c commit.gpgSign=false`.

### Queue and timing

- **R3** — Each write MUST add its paths and a `(tool, title, path)` entry to the
  queue without blocking the tool call.
- **R4** — A batch MUST be committed when 2 s pass with no new entry, or 30 s
  after the first entry of the batch, whichever comes first.
- **R5** — One batch MUST be one commit. A batch of one entry MUST have the
  message `memory: <tool> <title>`. A batch of several MUST have the subject
  `memory: <n> changes` and a body with one line per entry,
  `- <tool> <title> (<path>)`, in queue order. Titles MUST have line breaks
  replaced by spaces.

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

### Shutdown (ADR-0014)

- **R12** — On the shutdown context (stdin EOF, SIGINT, SIGTERM), after the tool
  call in progress finishes, the committer MUST commit the pending batch at once,
  without waiting for the quiet window, and `memctl mcp` MUST exit only after
  that commit returns (success or failure).
- **R13** — A second SIGINT or SIGTERM during shutdown MUST exit at once with
  status 1, without waiting for the commit, after one `WARN` line.

## Error handling

| Failure | Result |
|---|---|
| Root not in a git work tree, or `git` missing | `WARN` at start; writes work, nothing is committed (R1) |
| `index.lock` present | Up to 3 retries, then `ERROR` (R10) |
| Hook fails, timeout, other git error | `ERROR` with the class; files stay uncommitted (R10) |
| Bad `--git-author` | Usage error, status 2 (SPEC-0004 R1) |

## Acceptance criteria — "Done looks like"

- [ ] **R1, R2** — A root outside git, a subfolder of a repository, and a `PATH`
  without `git`; a fake `git` on `PATH` records only the allowed subcommands and
  options.
- [ ] **R3, R4** — With a fake clock: one write commits after 2 s; writes every
  second commit at 30 s; tool calls return before the commit.
- [ ] **R5** — Messages for one and three entries; a title with a line break.
- [ ] **R6** — With a person's staged change in another file, a tool commit
  contains only the tool's paths, and the person's change stays staged; a move
  and a delete are committed as such; an ignored path is dropped.
- [ ] **R7** — In a repository with no `user.name`, the commit succeeds with
  the default author and committer; `--git-author "Ann <ann@example.org>"` is
  used when given.
- [ ] **R8** — A failing `pre-commit` hook gives one `ERROR` with `hook_failed`.
- [ ] **R9** — An edit reverted before the batch commits gives no commit.
- [ ] **R10, R11** — A held `index.lock` released after 2 s: commit after a retry;
  held for 10 s: `ERROR` `index_locked`; log lines hold no title or path.
- [ ] **R12** — An end-to-end test writes a note and closes stdin at once; the
  commit exists when the process has exited.
- [ ] **R13** — A second SIGTERM during a slow fake commit exits at once with
  status 1.

## References

- Design: D5, D10, D12, D13, D21, "Git" in
  [`../../design/solution-design.md`](../../design/solution-design.md).
- [ADR-0014](../../adr/0014-shutdown-flushes-pending-commits.md).
- SPEC-0004 (`memctl mcp`), SPEC-0005 (`memctl check`), SPEC-0007 (write tools).
