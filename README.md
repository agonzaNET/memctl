# memctl

Local Markdown memory for AI agents: a single-binary Go MCP server with
full-text search, a link graph and git-backed history.

Your notes stay plain Markdown files in one folder. `memctl` serves them to an
MCP client such as Claude Code or Claude Desktop, so an agent can search,
read, write, edit, move and delete notes, and follow the links between them.
Every change a tool makes is a git commit in the notes folder. Nothing leaves
your machine: there is no server, no account and no network access.

**Status:** pre-1.0 (`0.x`). The tools and the note format may still change
between minor versions; see [`CHANGELOG.md`](CHANGELOG.md).

## Install

Download the archive for your machine from the
[releases](https://github.com/agonzaNET/memctl/releases):
`memctl_<version>_linux_amd64.tar.gz`, `memctl_<version>_linux_arm64.tar.gz`
or `memctl_<version>_darwin_arm64.tar.gz` (Apple silicon), and `SHA256SUMS`.

```sh
sha256sum -c --ignore-missing SHA256SUMS      # macOS: shasum -a 256 -c --ignore-missing SHA256SUMS
tar -xzf memctl_<version>_<os>_<arch>.tar.gz
install -m 0755 memctl ~/.local/bin/          # or any folder on your PATH
memctl version
```

On macOS, a binary downloaded with a browser carries a quarantine attribute,
and Gatekeeper refuses to start it because it is not signed. Remove the
attribute once: `xattr -d com.apple.quarantine ~/.local/bin/memctl`.

Windows is not supported.

## Use it from an MCP client

`memctl mcp` is started by the client as a child process and talks MCP over
stdin and stdout. Point `--root` at your notes folder; it should be a git
repository (`git init` it once) so that every change is recorded.

Claude Code:

```sh
claude mcp add --scope user memory -- memctl mcp --root ~/notes
```

Claude Desktop (`claude_desktop_config.json`; use absolute paths, `~` is not
expanded there):

```json
{
  "mcpServers": {
    "memory": {
      "command": "/Users/you/.local/bin/memctl",
      "args": ["mcp", "--root", "/Users/you/notes"]
    }
  }
}
```

Only one `memctl mcp` can serve a notes folder at a time. A second client
configured on the same folder fails to start with
`root <path> is in use by memctl (pid N)`; give each client its own folder.

### Flags

| Flag | Environment variable | Default | Meaning |
|---|---|---|---|
| `--root DIR` | `MEMCTL_ROOT` | (required) | The notes folder |
| `--log-level LEVEL` | `MEMCTL_LOG_LEVEL` | `info` | `debug`, `info`, `warn` or `error`; JSON logs on stderr |
| `--git-author "Name <email>"` | `MEMCTL_GIT_AUTHOR` | `memctl <memctl@localhost>` | Author and committer of the tool commits |

A flag wins over its environment variable.

### Tools

| Tool | What it does |
|---|---|
| `search` | Full-text search (BM25), with type, tag and folder filters; identifiers such as `IT-104` or `10.50.4.15` match whole |
| `note_read` | A note, or one section of it, with its links and its `sha256` |
| `context` | The notes linked to and from a note, up to three hops, with every link between them |
| `list` | Notes by folder, type or tag, with tags and a date |
| `note_write` | Create a note from a title and a body |
| `note_edit` | Append, replace a section, or replace one piece of text; needs the `sha256` from `note_read`, so a change made meanwhile in an editor is never overwritten |
| `note_move` | Move or rename a note; the old path and title go into `aliases`, so links keep working |
| `note_delete` | Delete a note; git history keeps it |

## Notes

A note is a Markdown file with optional YAML frontmatter:

```markdown
---
title: PAN-OS upgrade procedure
type: runbook
tags: [palo-alto, firewall]
aliases: [PAN-OS upgrade]
created: 2026-11-19T10:12:00+01:00
updated: 2026-11-19T10:12:00+01:00
---

# PAN-OS upgrade procedure

Free text. A link such as [[hosts/PA-850]] is indexed as links_to.

## Observations
- [risk] HA pair must be upgraded passive first

## Relations
- applies_to [[hosts/PA-850]]
```

- Link to a note by its path without `.md` (recommended, always unique), by
  its title, or by one of its aliases.
- Observations (`- [category] text`) and relations (`- type [[Target]]`) are
  read only under the `## Observations` and `## Relations` headings.
- Files and folders whose names start with `.`, and what your `.gitignore`
  files exclude, are not notes. `memctl` keeps its own files in `.memory/`,
  which git ignores.
- Edit notes in any editor at any time; the next tool call sees the change.

The full format is in the [design](docs/design/solution-design.md#note-format-v1).

## Git

Each tool write is committed in the background, a few seconds after the last
write of a burst, as `memory: <tool> <path>` or `memory: <n> changes`. Only
the files a tool changed are committed: your own staged changes to other
files are left alone. A note you change in an editor, or while `memctl` is
not running, is committed on its own as `memory: external change <path>`,
with the repository's git identity when it has one, so your edits and the
tools' edits stay apart in the history. One exception: an edit you make to a
note in the few seconds between a tool's write to it and that write's commit
shares one commit with it. Pending changes are committed before `memctl mcp`
exits.
`memctl` never pushes, pulls or changes branches. A notes folder outside git
works too, without commits.

## Check a notes folder

```sh
memctl check --root ~/notes
```

Reports frontmatter and structure problems, unsupported Markdown, dangling,
ambiguous and stale links, skipped files and a missing git repository. It
exits 1 when there are errors. It reads only, and can run while `memctl mcp`
serves the same folder.

## Building from source

```sh
mise install    # the pinned Go toolchain
make build      # bin/memctl for this machine
make verify     # tests, vet, gofmt, the docs checks and the script tests
make dist VERSION=0.1.0   # the release archives in dist/
```

See [`docs/agents/environment.md`](docs/agents/environment.md).

## Design and decisions

- [`docs/design/`](docs/design/) — the product design (the source of truth).
- [`docs/adr/`](docs/adr/) — architecture decision records (the why).
- [`docs/specs/`](docs/specs/) — capability specs (the what).
- [`CHANGELOG.md`](CHANGELOG.md) — notable changes per release.

## License

[MIT](LICENSE)
