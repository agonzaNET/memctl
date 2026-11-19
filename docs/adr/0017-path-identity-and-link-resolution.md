# ADR-0017 - Identify notes by path, resolve links path first, report ambiguity, and keep old paths as aliases on a move

## Status

Accepted

**Area:** Index

## Context

The first draft of the design identified a note by a unique title: `note_write`
refused a title that existed, links resolved by title, then alias, then file
stem, and `memctl check` reported every repeated title.

Real notes do not fit that. Notes written for Basic Memory under a path-link
convention link by relative path (`[[network/intpxy]]`), and two notes in
different folders may share a title (a host and a service both called
`intpxy`). A title-only link to a shared title has no right answer; Basic
Memory resolves it to no note without an error, which hides broken links.

A move must not break those path links either, and the design already rules
out rewriting other notes on a rename (D16): a mass rewrite touches many files
and can overwrite a note open in an editor.

## Decision

> The relative path is a note's identity and its only unique key. A link or a
> `ref` resolves by path, then title, then alias, then file stem; at the first
> step that finds any note, one match resolves and several give `ambiguous`.
> `note_move` keeps old links working by appending the old path (and the old
> title, if it changes) to `aliases`, and never edits other notes.

- **Path step.** The target is compared with every note path, lower-cased,
  with or without `.md`, after removing a leading `./`. Aliases that look like
  paths (contain `/` or end in `.md`) are normalised the same way and match in
  the alias step.
- **No fall-through on ambiguity.** If the title step finds two notes, the
  alias and stem steps are not tried; the link is ambiguous.
- **Reporting.** A tool `ref` that is ambiguous fails with the code
  `ambiguous` and lists the matching paths. A link that is ambiguous or
  dangling is reported by `memctl check` and shown as such in `note_read` and
  `context`; it never fails the call that reads it.
- **Titles may repeat.** `note_write` fails only when the path exists. When the
  title is already used, it succeeds and names the other notes, suggesting a
  path link.
- **Stale links.** A link that resolves only through an old path in `aliases`
  is reported by `memctl check` as stale, so it can be updated by hand.

## Alternatives Considered

| Alternative | Reason not chosen |
|---|---|
| Unique titles (the first draft) | Existing notes that share titles across folders would have to be renamed before they could be used. |
| Pick one note for an ambiguous name (first found, newest) | A wrong link that looks right; the error stays hidden. |
| Rewrite links in other notes on a move | Many files changed by one call, and an edit open in an editor can be lost. |
| A generated permanent `id` per note | Rejected by D16: another field to keep in sync, and links by id are unreadable to people. |

## Consequences

- **Positive:** Path links never break on repeated titles, and resolve after a
  move.
- **Positive:** Ambiguity is always visible, to the agent and in `memctl check`.
- **Negative:** `aliases` collect old paths over time; they stay until someone
  updates the links and removes them.
- **Risk / Note:** Two paths that differ only in letter case collide in the
  path step and are reported as ambiguous.

## References

- Design D16, D23, D24 in [`../design/solution-design.md`](../design/solution-design.md).
