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
  over stdio, with the read-only tool `note_read` (spec: SPEC-0004).
- `memctl version` prints the version of the binary (spec: SPEC-0004).

[Unreleased]: https://github.com/agonzaNET/memctl/commits/main
