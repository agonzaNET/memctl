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
- `memctl mcp` serves a notes root from one process only: a second
  `memctl mcp` on the same root stops at once with
  `root <path> is in use by memctl (pid <N>)`. memctl keeps its own files in
  `<root>/.memory/`, which git ignores (spec: SPEC-0006).

[Unreleased]: https://github.com/agonzaNET/memctl/commits/main
