# SPEC-0007 - Write tools and atomic writes

## Status

Implemented

> Amended on 2026-11-20 for the findings of the live check
> ([`2026-11-20-live-check.md`](../../design/2026-11-20-live-check.md)),
> delivered by stage plan 0004: R16, R17, R23 and R24 changed; R29 added; the
> code `section_not_found` used by `note_edit`.

> Amended on 2026-11-23 by [SPEC-0011](../0011-instance-scope/spec.md)
> (ADR-0025): with a scope, each write tool's description ends with the
> scope line.

<!--
Status values: Draft | Accepted | Implemented | Superseded by SPEC-NNNN | Deprecated
Lifecycle: Draft (under review) → Accepted (approved; the code may be built,
but at least one acceptance criterion, for example a live test, is still
open) → Implemented (shipped; code satisfies this spec and all acceptance
criteria are ticked). Superseded by SPEC-NNNN and Deprecated are terminal.
-->

**Governing ADRs:** ADR-0002, ADR-0005, ADR-0011, ADR-0013, ADR-0015, ADR-0016, ADR-0017

## Purpose

Let an agent create, change, move and delete notes through four MCP tools —
`note_write`, `note_edit`, `note_move` and `note_delete` — without ever
overwriting a file in a state it has not seen, without leaving a half-written
file, and with every change handed to the git committer (SPEC-0008).

## Scope

**In scope**

- `internal/note`: the four write operations, the file-name rule, the atomic
  write, frontmatter changes (`created`, `updated`, `title`, `aliases`), index
  updates, and handing changed paths to the committer.
- `internal/mcpserver`: the four tools, their schemas, annotations,
  descriptions, results and error codes.

**Out of scope**

- The commit itself, its timing and author (SPEC-0008).
- The instance lock (SPEC-0006); writes assume it is held.
- Rewriting links in other notes on a move (design D24: never).
- Attachments and non-Markdown files.

## Inputs

| Tool | Argument | Type | Required | Rule |
|---|---|---|---|---|
| `note_write` | `title` | string | yes | 1–200 characters, no line break |
| | `folder` | string | no | relative folder, the root when absent; path rules of SPEC-0003 R14 |
| | `body` | string | yes | at most 1 MiB; may be empty |
| | `type` | string | no | 1–64 characters, no line break |
| | `tags` | array of string | no | at most 20, each 1–64 characters, no line break |
| `note_edit` | `ref` | string | yes | as `note_read` |
| | `expected_sha` | string | yes | 64 lower-case hex characters |
| | `op` | string | yes | `append`, `replace_section` or `find_replace` |
| | `text` | string | for `append`, `replace_section` | at most 1 MiB |
| | `section` | string | for `replace_section` | heading text |
| | `find` | string | for `find_replace` | 1 character to 64 KiB |
| | `replace` | string | for `find_replace` | at most 1 MiB; may be empty |
| `note_move` | `ref` | string | yes | as `note_read` |
| | `new_path` | string | at least one of the two | relative path ending in `.md`; SPEC-0003 R14 |
| | `new_title` | string | at least one of the two | as `title` |
| `note_delete` | `ref` | string | yes | as `note_read` |

## Outputs

ADR-0015 results. The first lines:

- `Created "<title>" at <path>.`
- `Edited "<title>" at <path>.`
- `Moved "<title>" from <old path> to <new path>.`
- `Deleted "<title>" at <path>.`

followed by `sha256: <hex>` of the new file (not for a delete), the
`Unresolved links` list of R29 when it applies, and the notices of the
requirements below. Each successful write hands its changed paths, the
tool name and the note title to the committer (SPEC-0008).

## Behaviour & Requirements

### Every write

- **R1** — Every write tool MUST run the freshness check and the operation under
  the note service mutex (SPEC-0004 R7), resolve `ref` like a read
  (SPEC-0004 R14), and pass every path through `note.SafePath`
  (SPEC-0003 R14).
- **R2** — Every file write MUST be atomic: write to a temporary file in the
  target folder whose name starts with `.` (so the walk skips it), `fsync` it,
  give it the mode of the file it replaces (0644 for a new file), `rename` it
  over the target, then `fsync` the folder. On any failure the temporary file
  MUST be removed and the target MUST be unchanged.
- **R3** — Just before the rename, the service MUST hash the target file again;
  if it changed since the freshness check, the write MUST stop with `conflict`
  (a human edit in that window is never overwritten).
- **R4** — After a successful write, the index entry MUST be updated directly
  (parse + `Put`, or `Remove`), so the next freshness check finds the file
  unchanged.
- **R5** — Every successful write MUST hand the changed paths (old and new for a
  move), the tool name and the note title to the committer; a failed write MUST
  hand nothing.
- **R6** — Missing folders of a target path MUST be created with mode 0755. Empty
  folders left behind by a move or delete MUST NOT be removed.
- **R7** — `created` and `updated` values MUST be written as local time with its
  offset, to the second (`2006-01-02T15:04:05-07:00`), from an injectable clock.

- **R28** — A target path that the walk would not treat as a note (SPEC-0003
  R3–R5: a dot-segment, or matched by a `.gitignore`) MUST give
  `invalid_argument` that names the reason; such a note would be invisible to
  every later call.

### Frontmatter changes

- **R8** — The service MUST change frontmatter by editing the frontmatter text,
  not by re-encoding all of it: a changed key's line (or block, for a list) is
  replaced in place, a new key is inserted before the closing delimiter, and
  every other line, comment and unknown key MUST stay byte for byte as it was.
  Lists are written in flow style (`[a, b]`) with values quoted only when YAML
  needs it.
- **R9** — A note without frontmatter, or with frontmatter that has the problem
  `bad_frontmatter` (SPEC-0001 R5), MUST NOT get frontmatter changes from
  `note_edit`; `note_move` on such a note, with `new_title` or `new_path`,
  MUST give `invalid_argument` that asks to fix the frontmatter first (a move
  must write `aliases`, R22, and keeps the title only by writing it, R20).

### `note_write`

- **R10** — The file name MUST be the title as written (existing notes are named
  this way, and path links name them), with every `/`, `\`, `:`, `*`, `?`,
  `"`, `<`, `>`, `|`, control character and leading `.` replaced by `-`, runs of
  spaces collapsed to one, trimmed, cut to 200 bytes on a character boundary,
  plus `.md`. The path is `folder/` plus the file name.
- **R11** — When the path exists, or a note exists whose path equals it ignoring
  letter case, `note_write` MUST give `exists` naming that path and suggesting
  `note_edit`.
- **R12** — A `body` that starts with a frontmatter delimiter line MUST give
  `invalid_argument` asking to pass `title`, `type` and `tags` as arguments.
- **R13** — The file MUST be the frontmatter (`title`, `type` when given, `tags`
  when given, `created`, `updated`, in that order, `created` = `updated` = now),
  a blank line, and `body`, ending in exactly one newline.
- **R14** — When other notes have the same title (`index.Titled`), the result
  MUST name their paths after the `sha256` line and suggest linking by path.

### `note_edit`

- **R15** — `expected_sha` MUST equal the SHA-256 of the file after the freshness
  check; otherwise the tool MUST give `conflict` with the current `sha256` and the
  current file content in a fence (as SPEC-0004 R15), so the agent can retry
  without another read.
- **R16** — `append` MUST add `text` as a new paragraph: the file's trailing
  blank lines and line breaks are dropped, then one blank line, then `text`
  without its leading blank lines, and the file MUST end in exactly one
  newline. A file that is empty, or holds only frontmatter, gets no extra blank
  line before the text beyond the one after the frontmatter. The description
  MUST say that `append` starts a new paragraph, and that `find_replace` or
  `replace_section` extends an existing paragraph or list.
- **R17** — `replace_section` MUST keep the heading line found as in SPEC-0004
  R16, and replace every line after it up to the next heading of the same or a
  higher level (or the end of the file) with one blank line and then `text`
  without its leading and trailing blank lines, ending in one newline and
  followed by a blank line when another heading follows. An empty `text` leaves
  the heading followed by one blank line. No matching heading MUST give
  `section_not_found` with the note's headings and the advice to add the
  section with `append`, starting `text` with the heading line.
- **R18** — `find_replace` MUST replace `find` with `replace` in the whole file,
  frontmatter included, when `find` occurs exactly once. Zero occurrences MUST
  give `no_match`; more than one MUST give `not_unique` with the count and the
  advice to include more surrounding text.
- **R19** — After the operation, `updated` MUST be set to now (R7, R8), unless
  R9 applies. An edit that leaves the file byte for byte unchanged (before
  `updated`) MUST NOT write, MUST NOT hand anything to the committer, and MUST
  succeed with the first line `"<title>" at <path> is unchanged.`

### `note_move`

- **R20** — At least one of `new_path` and `new_title` MUST be given; neither
  MUST give `invalid_argument`. With `new_title` only, the new path is
  the old folder plus the file name of R10 for the new title. With `new_path`
  only, the title MUST NOT change. With both, the new path is `new_path` and the
  title becomes `new_title`.
- **R21** — A target path that exists (R11) MUST give `exists`. A target equal to
  the current path with no title change MUST succeed with the first line
  `"<title>" is already at <path>.` and write nothing.
- **R22** — The move MUST `rename` the file to its new path, then rewrite it
  atomically (R2) with the frontmatter changes: `title` set when it changed; the
  old path without `.md` appended to `aliases` when the path changed, and the
  old title when the title changed, each only when not already in `aliases`
  (design D24); `updated` set to now.
- **R23** — The move MUST NOT change any other note. The result MUST give the
  number of notes that link to the old path or old title and name up to 10 of
  their paths, sorted (`and <k> more` for the rest), and say that those links
  keep working through `aliases` and that `memctl check` lists them as
  stale.

### `note_delete`

- **R24** — `note_delete` MUST remove the file and its index entry. The result
  MUST give the number of notes whose links to it are now dangling and name up
  to 10 of their paths, sorted (`and <k> more` for the rest), and say that git
  history keeps the file.

### Links in a written note

- **R29** — After `note_write` and `note_edit`, the result MUST list the
  written note's relations and links whose target is `dangling` or `ambiguous`
  under `Unresolved links:`, in the format of the `Links` list of SPEC-0004 R15
  (with the candidates of an ambiguous one), so the agent can fix them at once.
  The list MUST be absent when every relation and link resolves. The log line
  MUST add the count `unresolved`.

### Tools

- **R25** — Annotations MUST be: `note_write` readOnly false, destructive false,
  idempotent false; `note_edit` readOnly false, destructive false, idempotent
  false; `note_move` readOnly false, destructive false, idempotent false;
  `note_delete` readOnly false, destructive true, idempotent true; all
  `openWorldHint` false.
- **R26** — Each description MUST state the side effect, that the change is
  committed to git in the notes root within a few seconds and is never pushed,
  that `note_edit` needs the `sha256` from `note_read`, and every error code the
  tool can return, under the rules of ADR-0013.
- **R27** — The tool-call log line MUST carry the counts `bytes` (written),
  `paths` (handed to the committer), and for `note_move` and `note_delete`
  `linkers`; never a title, path, find or replace text.

## Error handling

Codes added to the fixed list of SPEC-0004:

| Code | When | Tools |
|---|---|---|
| `exists` | Target path exists | `note_write`, `note_move` |
| `conflict` | `expected_sha` differs, or the file changed during the write | `note_edit` (all writes for R3) |
| `no_match` | `find` does not occur | `note_edit` |
| `not_unique` | `find` occurs more than once | `note_edit` |
| `section_not_found` | `section` matches no heading (code of SPEC-0004) | `note_edit` |

`invalid_argument`, `not_found`, `ambiguous` and `internal_error` keep their
meaning. A failed file operation is `internal_error` and states that the note
is unchanged.

## Acceptance criteria — "Done looks like"

- [x] **R1, R2** — A write interrupted after the temporary file (fault injected
  before `rename`) leaves the target unchanged and no temporary file.
- [x] **R3** — A file changed between the check and the rename (injected hook)
  gives `conflict` and keeps the human edit.
- [x] **R4, R5** — After each tool, the index equals a fresh load and the
  committer stub received the expected paths; a failed write sends none.
- [x] **R6, R7** — Writing into a new nested folder; the timestamps from a fixed
  clock.
- [x] **R28** — `note_write` and `note_move` into a folder that `.gitignore`
  matches give `invalid_argument`.
- [x] **R8, R9** — Golden tests: frontmatter with comments, unknown keys, block
  and flow lists, and quoted values, after `updated`, `title` and `aliases`
  changes; no frontmatter and bad frontmatter.
- [x] **R10, R11** — File names for titles with `/`, `:`, leading dot, long
  titles, multi-byte characters; an existing path and a case-only clash.
- [x] **R12, R13, R14** — `note_write` golden file; body with frontmatter; shared
  title notice.
- [x] **R15** — Stale `expected_sha` returns the current content and sha.
- [x] **R18, R19** — `find_replace` with zero and two matches, an edit inside
  the frontmatter, and a no-op edit.
- [x] **R16** — Golden tests: `append` to a file ending in no newline, one
  newline and three blank lines; text with leading blank lines; a file with
  only frontmatter; an empty file. The description names the paragraph rule.
- [x] **R17** — Golden tests: `replace_section` of a heading followed by a
  blank line and of one followed directly by text, text with leading and
  trailing blank lines, empty text, the last section of the file; a missing
  section gives `section_not_found` with the headings and the `append` advice.
- [x] **R20, R21, R22, R23** — Move by path, by title, by both, onto itself, onto
  an existing note; aliases after two moves; links to the old path still resolve
  (SPEC-0002 R5 `Stale`).
- [x] **R24** — A second delete gives `not_found`.
- [x] **R23, R24** — Move and delete with 3 linkers name all three; with 12,
  name 10 and `and 2 more`.
- [x] **R29** — `note_write` with a dangling and an ambiguous link lists both;
  with resolved links only, no list; `note_edit` that adds a dangling link
  lists it; the log line carries `unresolved`.
- [x] **Live check (again)** — The exit check of stage plan 0004 repeats the
  write checks of the live check of 2026-11-20 on a scratch root.
- [x] **R25, R26, R27** — Tool list annotations; `check-docs.sh` check 5 passes;
  captured log lines carry the counts and no argument value.
- [x] **Live check** — In a Claude Code session on a scratch root, create, edit,
  rename and delete notes; edit one in an editor between a read and an edit and
  see `conflict`. Recorded as a verification record.
  *(Run with Claude Desktop: [`2026-11-20-live-check.md`](../../design/2026-11-20-live-check.md).)*

## References

- Design: D5, D16, D23, D24, "MCP tools", "Tool writes" in
  [`../../design/solution-design.md`](../../design/solution-design.md).
- SPEC-0001 to SPEC-0006; SPEC-0008 (git committer) — planned.
- Live check of 2026-11-20, findings L1, L12, L13, L16, L17:
  [`2026-11-20-live-check.md`](../../design/2026-11-20-live-check.md).
