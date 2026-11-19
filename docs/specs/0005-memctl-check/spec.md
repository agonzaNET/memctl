# SPEC-0005 - `memctl check`

## Status

Accepted

<!--
Status values: Draft | Accepted | Implemented | Superseded by SPEC-NNNN | Deprecated
Lifecycle: Draft (under review) → Accepted (approved; the code may be built,
but at least one acceptance criterion, for example a live test, is still
open) → Implemented (shipped; code satisfies this spec and all acceptance
criteria are ticked). Superseded by SPEC-NNNN and Deprecated are terminal.
-->

**Governing ADRs:** ADR-0002, ADR-0005, ADR-0017

## Purpose

Report everything that is wrong or doubtful in a notes root, so a person can
fix it by hand before or while using `memctl mcp`: notes that parse badly,
Markdown forms that carry no structure, links that resolve to no note, to
several notes or only through an old path, files that were skipped, and a root
that is not a git repository. The command is read-only.

## Scope

**In scope**

- `memctl check [--root DIR]` in `cmd/memctl`, and the report built from
  `internal/note` (SPEC-0003), `internal/parse` problems (SPEC-0001),
  `index.Links()` (SPEC-0002) and a git repository test.
- The text report on stdout and the exit status.

**Out of scope**

- Fixing anything; a `--fix` mode is a later decision.
- Per-type frontmatter validation (design backlog).
- Uncommitted changes and git history.

## Inputs

| Flag | Environment variable | Default | Rule |
|---|---|---|---|
| `--root` | `MEMCTL_ROOT` | none (required) | An existing directory (SPEC-0003 R1) |

`memctl check` takes no instance lock (design D12) and may run while
`memctl mcp` serves the same root.

## Outputs

One line per finding on stdout, then one summary line:

```text
decisions/odr-017.md:12: dangling_link: [[PA-850]] matches no note; create it or link to an existing note by path.
hosts/intpxy.md:30: stale_link: [[old/intpxy]] resolves only through an old path in aliases of network/intpxy.md; link to network/intpxy instead.
runbooks/upgrade.md:4: bad_field: created is not a date or an RFC 3339 timestamp.
.: no_git_repo: The notes root is not in a git repository; memctl will not commit tool writes. Run git init in it.
3 notes checked, 0 errors, 4 warnings.
```

## Behaviour & Requirements

- **R1** — `memctl check` MUST load the root with the same rules as `memctl mcp`
  (SPEC-0003 R1–R9) and MUST NOT write, create, lock or change anything in it.
- **R2** — It MUST report, as findings:
  1. every problem of every note from the parser (SPEC-0001, nine codes);
  2. from `index.Links()`: `dangling_link` for a dangling target,
     `ambiguous_link` for an ambiguous one (the candidate paths in the message),
     `stale_link` for a link resolved through an old path (the resolved note in
     the message);
  3. `skipped_file` for each note skipped as too large or unreadable, with the
     reason;
  4. `no_git_repo` once, when the root is not inside a git work tree, and
     `git_missing` instead when the `git` command cannot be run.
- **R3** — Each finding MUST have a severity:

  | Severity | Codes |
  |---|---|
  | error | `bad_frontmatter`, `invalid_utf8`, `ambiguous_link`, `skipped_file` |
  | warning | every other code |

- **R4** — Each finding MUST be printed as
  `<path>:<line>: <code>: <message>`, with `.` as the path and no line for
  root-level findings, where the message is one sentence that says what is wrong
  and how to fix it. The finding's text MUST quote at most the link target or
  the frontmatter key, never other note text.
- **R5** — Findings MUST be sorted by path, then line, then code, with root-level
  findings last, followed by the summary line
  `<n> notes checked, <e> errors, <w> warnings.` (singular forms for 1).
- **R6** — The exit status MUST be 0 when there is no error-severity finding, 1
  when there is at least one, and 2 for a usage error (as SPEC-0004 R1). A root
  that cannot be opened MUST give status 1 after one `ERROR` log line on
  stderr, like `memctl mcp` (SPEC-0004 R2).
- **R7** — The git test MUST run `git -C <root> rev-parse --is-inside-work-tree`
  with a 5 s timeout and treat `true` on stdout as a git repository.
- **R8** — Logs MUST go to stderr as JSON at `warn` level and above; stdout MUST
  hold only the findings and the summary.

## Error handling

| Failure | Result |
|---|---|
| Usage error | Usage on stderr, status 2 |
| Root missing or unreadable | `ERROR` log line, status 1 |
| `git` not installed or times out | `git_missing` warning; the check goes on |
| A note unreadable or too large | `skipped_file` error; the check goes on |

## Acceptance criteria — "Done looks like"

- [ ] **R1** — The root's files, their modification times and `git status` are
  unchanged after a check, also on a root that `memctl mcp` holds open.
- [ ] **R2, R3, R4, R5** — A fixture root with one example of every code gives a
  golden report, byte for byte.
- [ ] **R6** — Exit statuses: clean root 0; only warnings 0; one error 1; bad flag
  2; missing root 1.
- [ ] **R7** — In a git repository, in a subfolder of one, outside one, and with
  `PATH` that has no `git`.
- [ ] **R8** — stdout of a check holds only finding lines and the summary.

## References

- Design: D12, D17, D23, D24, "Deployment and config" in
  [`../../design/solution-design.md`](../../design/solution-design.md).
- SPEC-0001, SPEC-0002, SPEC-0003, SPEC-0004.
