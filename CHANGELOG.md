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
At a release, rename this section to `## [X.Y.Z] - YYYY-MM-DD`, open a fresh
`## [Unreleased]` above it and move this comment up. See
docs/agents/changelog.md.
-->

### Added

- `make mcpb MCPB_NAME=<name>` builds a Claude Desktop extension
  (`dist/<name>_<version>_darwin_arm64.mcpb`) holding the memctl binary.
  Opening it installs memctl in Claude Desktop and asks for the notes folder,
  which can be changed later in the extension's settings. Build one bundle
  per notes folder; the name is the server name Desktop shows. Needs Node.js
  (ADR-0024).

## [0.3.2] - 2026-11-22

The first release with the changes of 0.3.0, which was tagged but not
released. Coming from 0.2.0, read
[0.3.0 in CHANGELOG.md](https://github.com/agonzaNET/memctl/blob/v0.3.2/CHANGELOG.md#030---2026-11-21)
too.

### Fixed

- The `git` processes that `memctl mcp` starts no longer run with SIGPIPE
  ignored, and with git before 2.54 neither do the repository's hooks
  (spec: SPEC-0009).
- A second `memctl mcp` whose client ends normally no longer, at times, logs
  `memctl (pid N) serving the root stopped` and exits with status 1
  (spec: SPEC-0009).

## [0.3.1] - 2026-11-22

Tagged but not released: a test failed on macOS. Its changes ship in 0.3.2.

## [0.3.0] - 2026-11-21

Tagged but not released: the release workflow failed. Its changes ship in
0.3.2.

### Added

- Several MCP clients on one machine can use the same notes folder at the
  same time, such as Claude Desktop's chats and its Cowork and Code
  sessions: the first `memctl mcp` serves the folder, and every later one
  passes its client's messages to it over a socket in `.memory/`. A later
  process with a different `--git-author` is refused (spec: SPEC-0009).

### Changed

- A second `memctl mcp` on a folder no longer stops at once: it stops only
  when no shared session starts within 10 s, with
  `root <path> is in use by memctl (pid N), and it did not accept a shared session within 10 s`
  (spec: SPEC-0006, SPEC-0009).
- The first process stops when its own client and every other client have
  ended, or on a signal, and commits what is queued first (spec: SPEC-0004,
  SPEC-0008, SPEC-0009).
- The `tool call` log line names its session, `stdio` or `forward-<n>`; the
  `started` line says whether `sharing` is on (spec: SPEC-0004, SPEC-0009).
- Releases run the test suite on macOS (Apple silicon) for the tag before
  the binaries are built, and the macOS tests can be started by hand for any
  branch (ADR-0022).

## [0.2.0] - 2026-11-20

### Added

- `context` takes `edges`: `start` lists the same notes with only the
  connections to and from the start note, and names for each note further
  than one hop the note it was reached through (`via`), so a walk at depth 2
  or 3 stays small; `all`, the default, lists every connection between the
  notes as before (spec: SPEC-0002, SPEC-0004).
- `context` shows the number of connections in the heading of its list,
  `Edges (<m>):` (spec: SPEC-0004).
- A `search` that finds no notes names words in the notes that are a query
  word followed by digits, such as `cisco1` for `cisco`, to search for
  instead (spec: SPEC-0002, SPEC-0004).

### Changed

- A `search` hit on a line longer than 200 characters shows the part of the
  line around the first matching word, cut at whole words, with `…` on each
  side that was cut, instead of the first 200 characters (spec: SPEC-0002,
  SPEC-0004).
- The suggestions for an unknown ref include near names: a word such as
  `cisco` also finds notes that hold `cisco1` or `cisco12`. `search` ranking
  does not change (spec: SPEC-0002, SPEC-0004).

### Security

- Built with `golang.org/x/sys` v0.50.0, which fixes GO-2026-5024; memctl
  does not call the affected code, and `govulncheck` now reports no
  vulnerability at all.

## [0.1.0] - 2026-11-20

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
  `SHA256SUMS`, published by a manually started GitHub Actions workflow that
  stops when `govulncheck` finds a vulnerability the code calls; the README
  covers installing them and setting up Claude Code and Claude Desktop
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
  with the repository's git identity when it has one (`user.name` and
  `user.email`; a warning at start names a missing one), and never folded
  into a tool's commit (spec: SPEC-0003, SPEC-0008).
- `memctl mcp --git-author "Name <email>"` (or `MEMCTL_GIT_AUTHOR`) sets the
  author of those commits; the default is `memctl <memctl@localhost>`
  (spec: SPEC-0008).

[Unreleased]: https://github.com/agonzaNET/memctl/compare/v0.3.2...HEAD
[0.3.2]: https://github.com/agonzaNET/memctl/compare/v0.3.1...v0.3.2
[0.3.1]: https://github.com/agonzaNET/memctl/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/agonzaNET/memctl/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/agonzaNET/memctl/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/agonzaNET/memctl/releases/tag/v0.1.0
