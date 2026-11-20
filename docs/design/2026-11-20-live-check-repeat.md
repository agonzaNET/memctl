# Live check repeated after stage plan 0004 — 2026-11-20

What was checked: the exit check of stage plan 0004 (T16). The scripted
prompt of the first live check
([`2026-11-20-live-check.md`](2026-11-20-live-check.md)) was repeated, with
four checks added: identifier search (R5), an editor change between two tool
writes (W8), an edit while memctl was stopped (W9) and a note path that looks
like a pattern (W10). The maintainer ran it on macOS (Apple silicon) with
Claude Desktop and `bin/memctl` built from commit `6cfa17c` (T1–T15 and
T17–T19). Two servers were configured:

- `memctl-infra` on a copy of the maintainer's notes root, used for reads only;
- `memctl-live` on `/tmp/memctl-live`, reset to an empty git repository before
  the run.

## Results

| Check | Result | Evidence |
|---|---|---|
| R1 `context` at depth 2 | Pass | `Found 32 notes within 2 hops of "Upgrade PAN-OS on the internet-facing firewalls", plus the note itself.`; the edges between two notes at depth 2 are listed; a body link and a relation to one target appear once; two spot-checked notes match their edges |
| R2 `list` by folder and by type | Pass | `Listed 3 notes.`, `Listed 12 notes.`, sorted by path; every line has tags and `file changed 2026-11-20` |
| R3 Refs | Pass | `resolved by: title "EU6 firewall"`, `resolved by: alias "PAN-OS upgrade"`; an unknown ref gives `not_found` with three closest paths |
| R4 Editor edit seen on the next call | Pass | New `sha256`; `search` for `LIVECHECK-FRESH-2` gives `Found 1 note` |
| R5 Identifier search | Pass | `Found 4 notes for "IT-104".`, all four hold `IT-104`; `palo-alto` with limit 3 gives `Found 15 notes for "palo-alto"; showing the first 3.` and the closing line; tags in every hit |
| W1 `note_write` | Pass | `Created "Beta" at Beta.md.` with `Unresolved links:` `- links_to → Alpha: dangling` |
| W2 `context` on new notes | Pass | `Found 2 notes within 2 hops of "Alpha", plus the note itself.` with the three edges |
| W3 `note_edit` | Pass | `append` starts a paragraph after a blank line; `replace_section` keeps one blank line after the heading; `find_replace` correct |
| W4 `note_move` | Pass | `Moved "Gamma Renamed" from Gamma.md to archive/Gamma Renamed.md.` `1 note links to it: Alpha.md.`; `Gamma` resolves by alias |
| W5 `note_delete` | Pass | `Deleted "Beta" at Beta.md.` `1 note links to it: Alpha.md.`; Alpha shows the link as dangling |
| W6 Conflict | Pass | `conflict: …` with the current `sha256` and the editor line; the editor change got its own commit `memory: external change Alpha.md` |
| W7 Errors | Pass | `exists: …`; `section_not_found: … its headings are: Context, Decision; …` for `note_edit` and `note_read`; `other notes with this name: archive/Alpha.md` |
| W8 Editor change between two tool writes | Pass | Three commits in order: `note_edit Alpha.md`, `external change Alpha.md`, `note_edit Alpha.md` |
| W9 Edit while memctl was stopped | Pass | After the restart, `memory: 2 external changes` with `Delta.md` and `archive/Gamma Renamed.md`; clean working tree |
| W10 Literal note path | Pass | `Created "[draft] x" at [draft] x.md.`; the commit holds only `A [draft] x.md`. No other file was dirty, so the "never stage another file" half is proven by `TestCommitter_SPEC0008_R2_literalPaths`, not by this run |
| M1 Git history | Pass | One commit per change, a path in every subject, the move as `Gamma.md → archive/Gamma Renamed.md`, `git status --short` empty |
| M2 Second instance | Pass | `root /tmp/memctl-live (/private/tmp/memctl-live) is in use by memctl (pid 34549)`, exit 1; the plain line and the `ERROR` line in a terminal |

Every finding of the first live check marked for stage plan 0004 (C1, C2,
C4, L1 to L13, L15 to L17) is fixed. L14 (slow `note_read` calls) did not
recur: no delay in about 35 calls.

## Findings

| ID | Finding | Kind | Settled in |
|---|---|---|---|
| N1 | The repository had `user.name` but no `user.email`, so external changes were committed as `memctl <memctl@localhost>` (as SPEC-0008 R14 says), with nothing telling the person why | Spec gap | SPEC-0008 R17, stage plan 0004 T20: one `WARN` at start names the missing key; README states the rule |
| N2 | A `search` snippet cut at about 200 characters can end before the matched term, so the hit does not show why it matched | UX | Design, "Backlog for 0.x" |
| N3 | `context` at depth 2 on a real root returns 33 notes and about 170 edges: complete, but costly for an agent's context | UX | Design, "Backlog for 0.x" |
| N4 | Whole-identifier matching keeps `cisco` from matching `cisco1`, so `not_found` suggestions miss near names (SPEC-0002 R24, by design) | UX | Design, "Backlog for 0.x" |
| N5 | On a fresh copy every note shows `file changed` with the copy date | Informational | No change |

The maintainer decided on 2026-11-20 to fix N1 before the first release (v0.1.0)
and to leave N2 to N4 for a later 0.x version. T20 was verified by its Go
test and a sample run after this check.
