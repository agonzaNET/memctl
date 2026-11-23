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

## Why memctl

`memctl` is for knowledge you keep as documents: runbooks, decisions and
reference notes that you and your agents read, edit and link over time.

- **Not Claude's built-in memory.** Claude's memory holds short facts about one
  user, curated by Claude for its own conversations. `memctl` notes are plain
  files you own: you review every change in git, share the folder like any
  repository, and any MCP client can search and follow their links.
- **Not Claude Project knowledge.** Project files are uploads tied to
  claude.ai. `memctl` notes live on your machine, stay editable in any editor,
  and form a link graph an agent can walk.
- **Not Basic Memory.** `memctl` keeps Basic Memory's idea, Markdown notes
  with observations and relations, in one static Go binary with no Python
  runtime, no database and no cloud code. The design explains the trade-offs
  in [Context and goals](docs/design/solution-design.md#context-and-goals).

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

Claude Desktop, as an extension: build a bundle (see
[Building from source](#building-from-source)) and open the `.mcpb` file;
Claude Desktop asks for the notes folder and an optional scope (see
[Several notes folders](#several-notes-folders)) and installs memctl with its
own binary. The bundle's name is the server name Desktop shows and the
prefix of its tools; for two notes
folders, build two bundles with different names. The extension starts
`memctl mcp` with the default log level and git author.

Claude Desktop, by hand (`claude_desktop_config.json`; use absolute paths,
`~` is not expanded there):

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

### Several notes folders

Each notes folder is served by its own `memctl mcp`, under its own server
name. To the agent, two instances look the same apart from that name. Give
each one a scope, one line of up to 200 characters that says what the folder
holds and what does not belong there:

```json
{
  "mcpServers": {
    "infradocs": {
      "command": "/Users/you/.local/bin/memctl",
      "args": ["mcp", "--root", "/Users/you/work-notes",
               "--scope", "Work documents of the IT infrastructure team. No personal notes."]
    },
    "albdocs": {
      "command": "/Users/you/.local/bin/memctl",
      "args": ["mcp", "--root", "/Users/you/personal-notes",
               "--scope", "My personal notes: family, home lab, hobbies. No work documents."]
    }
  }
}
```

memctl appends the scope as one line to its server instructions and to every
tool description, so the agent reads it whichever of the two its client
passes on:

```text
Scope of this memory, set by the person who runs it: My personal notes: family, home lab, hobbies. No work documents.
```

The scope guides the agent; it does not stop a note from being written to
the wrong folder. With Claude Code, memctl's flag goes after `--`; the
`--scope user` before it is Claude Code's own:

```sh
claude mcp add --scope user notes -- memctl mcp --root ~/notes --scope "My personal notes."
```

### Several clients on one notes folder

Several clients on one machine can use the same notes folder at the same
time: Claude Desktop's chats and its Cowork and Code sessions (which start
one `memctl mcp` each from a single entry), or Claude Code and Claude
Desktop. The first `memctl mcp` on a folder serves it; every later one passes
its client's messages to the first over a socket in the folder's `.memory/`
(`memctl.sock`, readable by your user only). All clients share one index and
one git history, and every commit carries the first process's
`--git-author`.

- A later process with a different `--git-author` is refused, so a commit
  never carries an author you did not configure for that client.
- Every client gets the first process's `--scope`. A later process with a
  different scope is still served; both processes log `scope differs`.
- When the first process is stopped, the others end too; a client that
  restarts its server, as Claude Desktop does, gets one again, and the first
  to start serves the folder.
- If no shared session starts within 10 s, the process stops with
  `root <path> is in use by memctl (pid N), and it did not accept a shared session within 10 s`.
  An older `memctl` (before 0.3.0) on the folder gives this message, and so
  does a folder whose socket path is longer than your system allows (about
  100 characters); the first process then logs `sharing is off`.
- Each `tool call` log line names its session: `stdio` for the first
  process's own client, `forward-1`, `forward-2`, … for the others.

### Flags

| Flag | Environment variable | Default | Meaning |
|---|---|---|---|
| `--root DIR` | `MEMCTL_ROOT` | (required) | The notes folder |
| `--log-level LEVEL` | `MEMCTL_LOG_LEVEL` | `info` | `debug`, `info`, `warn` or `error`; JSON logs on stderr |
| `--git-author "Name <email>"` | `MEMCTL_GIT_AUTHOR` | `memctl <memctl@localhost>` | Author and committer of the tool commits |
| `--scope TEXT` | `MEMCTL_SCOPE` | none | One line of up to 200 characters telling the agent what the folder holds ([Several notes folders](#several-notes-folders)) |

A flag wins over its environment variable.

### Tools

| Tool | What it does |
|---|---|
| `search` | Full-text search (BM25F), with type, tag and folder filters; identifiers such as `IT-104` or `10.50.4.15` match whole; each hit shows the matching line, cut around the first match when long; without hits, it names near names such as `cisco1` for `cisco` |
| `note_read` | A note, or one section of it, with its links and its `sha256` |
| `context` | The notes linked to and from a note, up to three hops, with every link between them, or with `edges` `start` only the links of the start note and, for each farther note, the note it was reached through |
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
tools' edits stay apart in the history. That identity needs both `user.name`
and `user.email` (`git config` in the notes folder, or your global git
config); when one is missing, `memctl mcp` logs a warning at start that names
it, and your edits are committed with the `--git-author` identity instead.
One exception: an edit you make to a note in the few seconds between a tool's
write to it and that write's commit shares one commit with it. Pending
changes are committed before `memctl mcp` exits.
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
make mcpb MCPB_NAME=notes # a Claude Desktop extension (macOS, Apple silicon) in dist/; needs Node.js
```

See [`docs/agents/environment.md`](docs/agents/environment.md).

## Design and decisions

- [`docs/design/`](docs/design/) — the product design (the source of truth).
- [`docs/adr/`](docs/adr/) — architecture decision records (the why).
- [`docs/specs/`](docs/specs/) — capability specs (the what).
- [`CHANGELOG.md`](CHANGELOG.md) — notable changes per release.

## License

[MIT](LICENSE)
