# Live check of M1 and M2 with Claude Desktop — 2026-11-20

What was checked: the open live checks of SPEC-0004 (read tools on a copy of a
real notes root) and of SPEC-0007 and stage plan 0002 T13 (writes, conflict,
git history, instance lock), run by the maintainer on macOS (Apple silicon)
with `bin/memctl` built by `make build` at commit `a7e94ce`. The MCP client was
Claude Desktop, not Claude Code as the spec and the plan say: Claude Code was
not installed, and both clients start memctl the same way, as a stdio child
process. Two servers were configured:

- `memctl-infra` on `/tmp/memctl-infra`, a copy of the maintainer's Basic
  Memory notes (58 notes) with one baseline commit;
- `memctl-live` on `/tmp/memctl-live`, an empty git repository.

A Claude Desktop session ran a scripted list of checks and paused for the
steps only the maintainer could do (editor changes, terminal commands). The
first read checks were run by hand in an earlier session.

## Results

| Check | Result | Evidence |
|---|---|---|
| `memctl check` on the copy | Pass | `58 notes checked, 0 errors, 645 warnings.`, most of them `observation_outside_section` (design D-level behaviour, "Follow-up after v1") |
| `search` with `type` and `tags` filters | Pass | `firewall` with type `runbook` gave the three runbooks; with tag `palo-alto`, 10 hits |
| `note_read` of a section | Pass | Only the section's lines; `sha256` of the whole file (SPEC-0004 R16, R17) |
| `context` at depth 2 | Pass | `Found 32 notes within 2 hops of "Upgrade PAN-OS on the internet-facing firewalls".`; two spot-checked notes exist and match their edges |
| `list` by folder and by type | Pass | `Listed 3 notes.`, `Listed 12 notes.`, sorted by path |
| Ref by title, by alias, unknown ref | Pass | Title and alias resolve; `not_found: No note matches "runbooks/Upgrade the Cisco switches"; use search to find the note, then pass its path.` |
| Editor edit seen on the next call | Pass | New `sha256`, new line present, `search` finds the new word with the note ranked first |
| `note_write`, `note_edit` (three ops), `note_move`, `note_delete` | Pass | `Created …`, `Edited …`, `Moved "Gamma Renamed" from Gamma.md to archive/Gamma Renamed.md.`, `Deleted "Beta" at Beta.md.`; a link to the old title resolves after the move; a link to the deleted note shows `dangling` |
| Conflict | Pass | `conflict: The note changed since you read it, so nothing was changed. …`, with the current content; the editor change stayed in the file |
| Errors name the next step | Pass | `exists: …; use note_edit to change it, or choose another title or folder.`, `not_found: The note has no heading "Consequences"; its headings are: Context, Decision.` |
| Git history | Pass | One commit per change, one batch commit (`memory: 3 changes`) for the first three writes, author `memctl <memctl@localhost>` |
| Second instance on the same root | Pass | `root /private/tmp/memctl-live is in use by memctl (pid 10659)`, at once |

Every check behaves as its spec says. Three checks of the session's report
were marked "fail" or "with findings" for behaviour the specs require as
written (append, `replace_section` spacing, context edges); they are spec gaps
and are listed below.

## Findings

C1–C4 were found in the first, manual session; L1–L17 are the numbered
findings of the scripted session's report.

| ID | Finding | Kind | Settled in |
|---|---|---|---|
| C1 | `search` and `list` do not show tags, so an agent cannot learn the tag values in use without reading notes | UX | SPEC-0004, stage plan 0004 |
| C2 | `search` does not say when it stopped at `limit`, so the agent guesses whether there are more hits | Tool text | SPEC-0002, SPEC-0004, stage plan 0004 |
| C3 | Scores are close together (0.41–0.47) for a one-word query | Watch | No change; design risk "BM25 ranking feels weak on short notes" |
| C4 = L7 | `note_read` with `section` lists the links of the whole note | UX | SPEC-0004, stage plan 0004 |
| L1 | `append` adds the text after one newline, so appended text joins the last paragraph (as SPEC-0007 R16 says) | Spec gap | SPEC-0007 R16, stage plan 0004 |
| L2 | `list` promises the last update date but prints it only when the note has `updated`; the copied notes have none | Tool text | SPEC-0004, stage plan 0004 |
| L3 | The tokenizer splits `IT-104`, `11.1.16-h2` and `10.50.4.15` into fragments (SPEC-0002 R7), so a search for an identifier is flooded by notes that match one fragment | Spec gap | Design (search), SPEC-0002 R7, stage plan 0004 |
| L4 | `context` lists only the edges it followed, so edges between two notes at the maximum depth are missing (SPEC-0002 graph walk) | Spec gap | Design D9, SPEC-0002, SPEC-0004, stage plan 0004 |
| L5 | The `context` count excludes the start note without saying so | Tool text | SPEC-0004, stage plan 0004 |
| L6 | `context` lists identical edges twice, and a body link and a relation to the same target as two edges | UX | SPEC-0002, SPEC-0004, stage plan 0004 |
| L8 | A ref resolved by title or alias does not say so; when a path and a title collide the path wins silently | UX | SPEC-0004, stage plan 0004 |
| L9 | `not_found` for a ref offers no candidates | UX | SPEC-0004, stage plan 0004 |
| L10 | An editor change to a note stays uncommitted until memctl next writes that note, and is then committed as memctl's change | Spec gap | Design (Git), SPEC-0008, stage plan 0004 |
| L11 | A one-change commit names the title, not the path; a move names only the new title. A batch's body already lists each change with its path (SPEC-0008 R5) | UX | Design (Git), SPEC-0008, stage plan 0004 |
| L12 | `replace_section` drops the blank line after the heading (SPEC-0007 R17) | Spec gap | SPEC-0007 R17, stage plan 0004 |
| L13 | `note_write` does not report dangling or ambiguous links in the new note | UX | SPEC-0007, stage plan 0004 |
| L14 | Three `note_read` calls took about 4 minutes to return | Not a memctl fault | No change: the tool-call log shows every call at 21 ms or less; the delay was the client's tool approval prompt |
| L15 | The lock refusal shows the resolved path (`/private/tmp/…`) and is printed twice (plain line and JSON log line) | UX | SPEC-0006, stage plan 0004 |
| L16 | `note_delete` reports only the number of notes that link to the deleted note | UX | SPEC-0007, stage plan 0004 |
| L17 | A missing section and a missing note share the `not_found` code | Tool text | SPEC-0004, SPEC-0007, stage plan 0004 |

The maintainer decided on 2026-11-20 to fix every finding marked for stage
plan 0004 before the first release (v0.1.0). The specs named above are amended
first; stage plan 0004 then plans them as one stage, and its exit check repeats
this live check.
