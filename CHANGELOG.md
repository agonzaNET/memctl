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
  links, or one section), `search` (BM25F full-text ranking with type, tag and
  folder filters), `context` (relations, links and backlinks up to three hops)
  and `list` (notes by folder, type or tag) (spec: SPEC-0004).
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
  `root <path> is in use by memctl (pid <N>)`. memctl keeps its own files in
  `<root>/.memory/`, which git ignores (spec: SPEC-0006).
- `note_write` creates a note from a title, a body, and optional type and
  tags, in a file named after the title; it refuses to overwrite an existing
  note (spec: SPEC-0007).
- `note_edit` changes a note by appending text, replacing one section, or
  replacing one exact piece of text, and sets `updated`. It takes the
  `sha256` from `note_read` and returns `conflict` with the current content
  when the note changed since that read, so an edit made in an editor is
  never overwritten (spec: SPEC-0007).
- `note_move` moves or renames a note. It never edits other notes: the old
  path and the old title go into the note's `aliases`, so links to them keep
  working, and `memctl check` lists them as stale (spec: SPEC-0007).
- `memctl mcp` picks up edits made in any editor at the next tool call,
  also a second save of the same size within one file-system timestamp tick:
  a note saved within 2 s of its last read is hashed again (spec: SPEC-0003).
- `note_delete` removes a note; the result counts the links that now point
  nowhere, and git history keeps the file (spec: SPEC-0007).
- Every tool write is committed to git in the notes root, in the background:
  one commit per burst of writes (after 2 s of quiet, or 30 s at most), only
  the files the tools changed, never pushed. Pending writes are committed
  before `memctl mcp` exits. A root outside git works without commits
  (spec: SPEC-0008).
- `memctl mcp --git-author "Name <email>"` (or `MEMCTL_GIT_AUTHOR`) sets the
  author of those commits; the default is `memctl <memctl@localhost>`
  (spec: SPEC-0008).

[Unreleased]: https://github.com/agonzaNET/memctl/commits/main
