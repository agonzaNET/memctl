# Changelog

All notable changes to memctl are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html). See
[`docs/agents/changelog.md`](docs/agents/changelog.md) for how to maintain this
file, and ADR-0007 for why it exists.

## [Unreleased]

<!--
Add entries as changes land, under the heading that applies:
Added · Changed · Deprecated · Removed · Fixed · Security
On the first release, rename this section to `## [X.Y.Z] - YYYY-MM-DD` and open a
fresh `## [Unreleased]` above it. See docs/agents/changelog.md.
-->

### Added

- `memctl mcp --root DIR` serves a folder of Markdown notes to an MCP client
  over stdio, with four read-only tools: `note_read` (a note with its resolved
  links, or one section with only its own links), `search` (BM25F full-text
  ranking with type, tag and folder filters, tags in each hit, and the total
  when more notes match than shown), `context` (relations, links and
  backlinks up to three hops, with every connection between the notes it
  lists) and `list` (notes by folder, type or tag, with tags and a date for
  every note). A ref says when it matched a title, alias or file name rather
  than a path, and names other notes with that name; an unknown ref gets up
  to three suggestions; a missing section has its own error,
  `section_not_found` (spec: SPEC-0002, SPEC-0004).
- `search` indexes identifiers such as `IT-104`, `10.50.4.15` or
  `11.1.16-h2` whole as well as in parts: searching an identifier finds only
  the notes that contain all of it, and searching one part (`104`) still
  finds it inside identifiers (spec: SPEC-0002).
- `memctl check --root DIR` reports parse problems, unsupported Markdown,
  dangling, ambiguous and stale links, skipped files and a missing git
  repository; it exits 1 when there are errors (spec: SPEC-0005).
- `memctl version` prints the version of the binary (spec: SPEC-0004).
- Release archives for Linux (amd64, arm64) and macOS (Apple silicon) with
  `SHA256SUMS`, published by a manually started GitHub Actions workflow; the
  README covers installing them and setting up Claude Code and Claude Desktop
  (ADR-0012).
- `memctl mcp` serves a notes root from one process only: a second
  `memctl mcp` on the same root stops at once with
  `root <path> is in use by memctl (pid <N>)`, naming the root as given and
  the resolved root when they differ. memctl keeps its own files in
  `<root>/.memory/`, which git ignores (spec: SPEC-0006).
- `note_write` creates a note from a title, a body, and optional type and
  tags, in a file named after the title; it refuses to overwrite an existing
  note. The result lists the note's links that match no note or several
  notes (spec: SPEC-0007).
- `note_edit` changes a note by appending a paragraph, replacing one section,
  or replacing one exact piece of text, sets `updated`, and lists the note's
  unresolved links. It takes the
  `sha256` from `note_read` and returns `conflict` with the current content
  when the note changed since that read, so an edit made in an editor is
  never overwritten (spec: SPEC-0007).
- `note_move` moves or renames a note. It never edits other notes: the old
  path and the old title go into the note's `aliases`, so links to them keep
  working, and `memctl check` lists them as stale (spec: SPEC-0007).
- `memctl mcp` picks up edits made in any editor at the next tool call,
  also a second save of the same size within one file-system timestamp tick:
  a note saved within 2 s of its last read is hashed again (spec: SPEC-0003).
- `note_delete` removes a note; the result names the notes whose links now
  point nowhere, and git history keeps the file. `note_move` names the notes
  that link to the moved note in the same way (spec: SPEC-0007).
- Every tool write is committed to git in the notes root, in the background:
  one commit per burst of writes (after 2 s of quiet, or 30 s at most), only
  the files the tools changed (a file name such as `[draft] x.md` is never
  read as a pattern), never pushed, with messages that name the
  paths (`memory: note_edit runbooks/x.md`). Pending writes are committed
  before `memctl mcp` exits. A root outside git works without commits
  (spec: SPEC-0008).
- Notes changed outside memctl, in an editor or while memctl was not
  running, are committed on their own as `memory: external change <path>`,
  with the repository's git identity when it has one, and never folded into a
  tool's commit (spec: SPEC-0003, SPEC-0008).
- `memctl mcp --git-author "Name <email>"` (or `MEMCTL_GIT_AUTHOR`) sets the
  author of those commits; the default is `memctl <memctl@localhost>`
  (spec: SPEC-0008).

[Unreleased]: https://github.com/agonzaNET/memctl/commits/main
