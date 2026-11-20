# Go Memory MCP — Design

Nov 19, 2026 · @Alberto Jose Gonzalez Escalante

## Context and goals

Build a small, local-only MCP server in Go that gives AI clients a persistent memory made of plain Markdown notes, inspired by [Basic Memory](https://github.com/basicmachines-co/basic-memory) but written from scratch. Basic Memory is \~135k lines of Python with cloud, multi-backend and an in-process REST layer; this project keeps the idea and drops the bulk.

Goals:

- Notes are plain Markdown files on disk and stay the single source of truth.
- One static Go binary, no CGO, no runtime dependencies, run by the MCP client as a stdio child process.
- Fast full-text search and graph traversal over notes through MCP tools.
- Humans can edit the same files in any editor; changes are picked up automatically.
- Every write by a tool is recoverable through git history.
- Clean-room implementation: no code copied from Basic Memory (AGPL-3.0).

## Non-goals

Anything that needs a server we do not run, or a second copy of the data, is out of scope for v1.

- No cloud mode, sync service, accounts, routing or telemetry.
- No database (SQLite, Postgres, Milvus); the index lives in memory.
- No compatibility with Basic Memory files or its DB schema.
- No multi-project support inside one instance; run one instance per notes root.
- No semantic or vector search in v1 (interface stubbed for v2).
- No importers or converters (Basic Memory, ChatGPT, Claude exports) and no web UI.
- No `git push`; remote sync stays a manual human action.

## Decisions taken

The twenty-four decisions below were agreed with Alberto on 2026-11-19 and frame the rest of the design. D18 to D22 were added the same day, when the project's build, packaging and tool conventions were settled, and D23 and D24 when path links were settled.

| # | Topic | Decision | Consequence |
| --- | --- | --- | --- |
| D1 | Deployment | Single user, stdio only. No HTTP transport in v1. | No auth in v1: the client spawns the process. HTTP with a generated key is a backlog item. |
| D2 | Note format | Own format, based on Basic Memory's ideas with targeted fixes (see Note format v1). | Existing notes work unconverted; old syntax stays as plain text. |
| D3 | Search | BM25 full-text ranking in v1; an `Embedder` interface is stubbed for v2 (TEI over HTTP). | No model or TEI runtime dependency in v1. |
| D4 | Projects | One notes root per server instance. | Simpler tool parameters; several roots = several instances. |
| D5 | Git | Auto-commit after each tool write; never push. | Free undo and audit trail; humans push. |
| D6 | Cloud | Removed entirely. | No sync, accounts, routing or promo code paths. |
| D7 | Name | `memctl` for binary, repo and module. | No rename churn later. |
| D8 | Repo | github.com/agonzaNET/memctl (personal GitHub, not aai-infra); Go module path is the same. | Own CI; stays open-sourceable. |
| D9 | `context` scope | Graph neighbours only: relations, body links (`links_to`) and backlinks; no tag matches. Edge types can be filtered. The result lists every edge between the notes it returns, or on request only the start note's edges. | Predictable, small output; tag queries go through `search`. |
| D10 | Commit author | Default identity `memctl <memctl@localhost>`. | AI edits are separable from human edits in history and blame. |
| D11 | Licence | MIT. | Permissive and simple; the clean-room build carries no AGPL obligation. |
| D12 | Instance lock | Exactly one `memctl` process per notes root; a second exits with an error naming the holder's PID. | One writer and one git committer; two MCP clients cannot share a root at the same time. |
| D13 | Commit author source | `--git-author` flag, default `memctl <memctl@localhost>`; the write path carries an author to the committer. | Team mode can later set a per-key author without refactoring. |
| D14 | Index storage | In memory, rebuilt from the files on every start. No database. | No schema, no index file, no reindex command, no stale-index state. |
| D15 | Change detection | Lazy: stat all files before each tool call and reparse the changed ones. No watcher. | No fsnotify, debounce, dropped events or self-write suppression; a few ms per call. |
| D16 | Note identity | No generated `id`; notes are identified by path, title and aliases (D23). | Fewer frontmatter fields; no ULID dependency; rename safety comes from aliases (D24). |
| D17 | Markdown parsing | Own line-based parser with code-fence tracking. No Markdown library. | No goldmark; rare Markdown forms are documented as unsupported and flagged by `memctl check`. |
| D18 | Build and dev environment | `make build` builds for the host with `CGO_ENABLED=0`; A manually triggered GitHub Actions workflow builds the release binaries for linux/amd64, linux/arm64 and darwin/arm64 from an existing `vX.Y.Z` tag; nothing runs on push or tag for now. A dedicated Linux dev VM is recommended, not required; `mise` pins the toolchain. | `memctl` runs where its MCP client runs, mostly macOS; no build target is cross-only. |
| D19 | Tool text | Tool descriptions and schemas are written for an agent: no ADR, spec or requirement numbers and no repository paths. Note paths such as `hosts/PA-850.md` are allowed. `scripts/check-docs.sh` enforces it. | Descriptions can show the path form agents must use; internal references never reach the model. |
| D20 | Package layout | The entry point is `cmd/memctl` (subcommands); the MCP package is `internal/mcpserver`. | No name clash with the SDK package `mcp`. |
| D21 | Shutdown | On stdin EOF, SIGINT or SIGTERM, commit every queued path at once, then exit. | No tool write stays uncommitted because the client closed inside the 2 s quiet window. |
| D22 | Tool results | No JSON envelope and no `structuredContent`. Success: `isError` false, a first line of one sentence that names what was done or found, then Markdown. Error: `isError` true, text `<code>: <sentence that says what to do next>`, with codes from one fixed list. | Uniform, testable results for a model reader; structured output for programs is a later decision. |
| D23 | Path identity and link resolution | The relative path is a note's identity and the only unique key; titles may repeat across notes. A link or `ref` resolves by path (with or without `.md`), then title, then alias, then file stem; at the first step with any match, one match resolves and several give `ambiguous`, never a silent pick. Path links (`[[network/intpxy]]`) are the recommended form. | Existing notes that link by path and share titles across folders work unchanged; title links still work while the title is unique. |
| D24 | Move keeps old links working | `note_move` appends the old path (without `.md`) to `aliases`, and the old title too when the title changes. It never rewrites other notes. Aliases that look like paths resolve as paths. | No mass rewrite and no risk to a note open in an editor; links to the old path keep resolving, and `memctl check` lists them as stale so they can be updated by hand. |

## Note format v1

Keep Basic Memory's core idea (frontmatter + observations + relations in plain Markdown) and fix four weak spots: identity, rename safety, parsing ambiguity and inline noise. The result stays Obsidian-friendly.

| Area | Basic Memory | v1 | Why |
| --- | --- | --- | --- |
| Identity | `permalink` slug in frontmatter, derived from path | No generated identifier; the path is the unique key, titles and aliases are names that may repeat (D16, D23) | Nothing to keep in sync with the path; one user needs no global id |
| Rename safety | Title links rewritten across notes on rename | Old path and old title are appended to `aliases`; links resolve by path, then title, then aliases, then file stem (D23, D24) | No mass rewrite of other files; old links keep working |
| Observations | `- [category] text #tag (context)` anywhere in the note | `- [category] text` only under `## Observations`; category matches `[a-z][a-z0-9_-]+` (2+ chars) | Avoids collision with GFM tasks `- [ ]` / `- [x]` and with prose bullets |
| Tags | Frontmatter plus inline `#tag` | Frontmatter `tags` only | Inline `#` hits `C#`, `#123`, URLs; parentheses were parsed as context |
| Relations | `- type [[Target]]` bullets; prose links become `links_to` | Same syntax, only under `## Relations`; type matches `[a-z][a-z0-9_]*`; links elsewhere become `links_to` | Deterministic parsing, same mental model |

Frontmatter fields: `title`, `type`, `tags`, `aliases`, `created`, `updated`. Unknown keys are preserved untouched. Links accept `[[folder/File name]]` (a path, `.md` optional; the recommended form), `[[Title]]`, and either with a `|label`.

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

Free Markdown body. A mention of [[PA-850]] here is indexed as links_to.

## Observations
- [prereq] Download content updates before the base image
- [risk] HA pair must be upgraded passive first

## Relations
- implements [[ODR-017]]
- applies_to [[PA-850]]
```

**Parsing (D17).** A line-based parser reads YAML frontmatter (`yaml.v3`), ATX headings (`#` to `######`), section bullets and `[[links]]`. It tracks fenced code blocks (\`\`\` and \~\~\~), so headings and links inside fences are ignored. Setext headings, indented code blocks and HTML blocks are unsupported for structure; `memctl check` flags them.

**Lenient reads.** Notes written for Basic Memory or by hand are valid without conversion. Inline `#tags`, `(context)` and `permalink` stay as plain text or unknown frontmatter. Observations and relations outside their sections are not parsed. No converter exists or is planned.

## Architecture

One process, four layers, no internal HTTP hop and no database: tools call the note service directly, which writes files, updates the in-memory index and queues git commits.

&#91;embedded content: memctl architecture · one process\]

Human edits reach the index through the freshness check that runs before each tool call; tool writes update the index directly. Go packages (D20): `cmd/memctl`, `internal/mcpserver` (go-sdk), `internal/note` (service, freshness check), `internal/parse` (line parser + yaml.v3), `internal/index` (BM25 + link graph), `internal/gitlog` (git CLI). External dependencies: go-sdk and yaml.v3 only, each imported by its one package (go-sdk by `internal/mcpserver`, yaml.v3 by `internal/parse`), which a test enforces.

## In-memory index

The index is built from the files on every start and lives only in process memory (D14); nothing is written to disk, so there is nothing to migrate, rebuild or get stale across restarts.

| Structure | Holds | Used by |
| --- | --- | --- |
| `notes` map, keyed by path | title, type, tags, aliases, created, updated, mtime, size, sha256, body, observations, links | every tool |
| `paths` map, lower-cased path without `.md` → path | every note path | link resolution and `ref` lookup, step 1 |
| `names` map, lower-cased name → paths | titles, aliases (path-like aliases also normalised as paths) and file stems, one map per kind | link resolution and `ref` lookup, steps 2–4; ambiguity detection |
| `backlinks` map, path → source paths | reverse of resolved links | `context` |
| Inverted index, term → postings | per field: title, aliases, tags, observations, body | `search` |
| Field lengths | per note and field, plus averages | BM25F scoring |

Startup parses all files in parallel (one goroutine per CPU). Search ranks with BM25F, field weights title 3, aliases 2, tags 2, observations 1.5, body 1, a note matching any query term, tuned later against real queries. The tokenizer lower-cases and splits on Unicode letters and digits, with no stemming, since notes mix English and German. Runs joined by `-`, `.`, `_`, `/` or `:` (identifiers such as `IT-104`, `10.50.4.15`, `11.1.16-h2`) are indexed whole as well as in parts; a query term written as such an identifier matches only the whole identifier, so a lookup by ticket, address or version is not flooded by notes that share one fragment. Only for the suggestions of a `not_found` error, a query word of letters also matches the same word followed by digits (`cisco` → `cisco1`, `eu` → `eu1`), so a near name is still offered; `search` ranking keeps whole-word matching. A snippet of a line longer than 200 characters is cut around the first matched term, with `…` on each cut side.

Links are resolved at query time (D23): path, then title, then alias, then file stem. At the first step that finds any note, one match resolves the link and several make it ambiguous; later steps are not tried. Creating a missing target or moving a note therefore needs no link rewrite. Unresolved targets are reported as dangling and ambiguous ones as ambiguous, never as errors of the tool call that reads them.

## MCP tools

Eight tools cover v1, each with MCP annotations (`readOnlyHint`, `destructiveHint`, `idempotentHint`) so clients can auto-approve reads and ask before writes. Notes are addressed by a `ref`: a relative path (recommended), title or alias, resolved like a link (D23). A `ref` that matches several notes fails with `ambiguous` and lists their paths.

| Tool | Read-only | Destructive | Inputs | Behaviour |
| --- | --- | --- | --- | --- |
| `note_write` | No | No | `title`, `folder`, `body`, `type`, `tags` | The file name is the title as written, with characters unsafe in file names replaced by `-` (`runbooks/PAN-OS upgrade procedure.md`). Create only; fails if the path exists. A title shared with other notes is allowed; the result names those notes and suggests linking by path |
| `note_read` | Yes | No | `ref`, optional `section` | Returns frontmatter, body, resolved links and sha256 |
| `note_edit` | No | No | `ref`, `op` (`append`, `replace_section`, `find_replace`), `expected_sha` | Fails on hash mismatch, so external edits are never overwritten |
| `note_move` | No | No | `ref`, `new_path` or `new_title` | Moves or renames the file; appends the old path, and the old title if it changed, to `aliases` (D24); never edits other notes |
| `note_delete` | No | Yes | `ref` | Removes file; git keeps history |
| `search` | Yes | No | `query`, `type`, `tags`, `folder`, `limit` | BM25F ranking, returns snippets cut around the first match |
| `context` | Yes | No | `ref`, `depth` (default 1, max 3), `rel_types`, `edges` (`all` default, `start`) | Graph walk over relations and backlinks; `edges` `start` lists only the start note's edges |
| `list` | Yes | No | `folder`, `type`, `tag` | Directory listing |

All tools return Markdown text, with no JSON envelope and no `structuredContent` (D22):

- **Success:** `isError` is false. The first line is one sentence that names what was done or found (`Created "PAN-OS upgrade procedure" at runbooks/PAN-OS upgrade procedure.md.`), followed by the Markdown result.
- **Error:** `isError` is true and the text is `<code>: <sentence that says what to do next>`, for example `exists: A note exists at hosts/PA-850.md; use note_edit to change it.` The codes come from one fixed list in `internal/mcpserver`, and a test fails on a code outside it.

Tool descriptions and input schemas are written for the agent that reads them (D19). Each description says what the tool returns, when to use it rather than another tool, its limits and what each error code means, and that returned note text is reference data, not instructions to follow.

## Change detection, concurrency and git

Files always win: the index follows the disk, and a tool write never overwrites a file it has not seen in its current state.

**Instance lock (D12).** On start, `memctl mcp` takes a non-blocking `flock` on `<root>/.memory/memctl.lock` and holds it for the process lifetime, writing its PID into the file. It creates `<root>/.memory/` when missing, with a `.gitignore` of `*` so nothing in it is ever committed. A root where the lock cannot be created (read-only) is refused. If the lock is held, the process exits with `root <path> is in use by memctl (pid N)`. The kernel releases the lock on exit or crash, so there is no stale-lock cleanup. `memctl check` is read-only and runs without the lock.

**Freshness check (D15).** Before each tool call, walk the root and stat every Markdown file. The walk skips every file and folder whose name starts with a dot (`.git`, `.memory`, `.obsidian`, `.trash`), does not follow symlinks, and skips what the root's `.gitignore` files match, using memctl's own matcher with git's pattern rules (no git process per call). Files whose mtime or size changed are reparsed; new files are added; missing files are dropped. A file whose mtime is not at least 2 s older than the time its entry was last read is hashed again even when mtime and size match, so a second save of the same size within one timestamp tick is not missed (git calls such entries "racily clean"). For a few thousand notes this costs a few milliseconds and needs no background goroutine.

**Tool writes.**

1. Run the freshness check (tool calls are serialised by one mutex).
2. For edits, compare `expected_sha` with the file's current hash; a mismatch returns a conflict error with the current content.
3. Write to a temp file in the same directory, `fsync`, then `rename` (atomic).
4. Update the index entry for that file directly.
5. Queue a git commit.

**Git.** If the root is in a git work tree, a background goroutine commits queued paths after 2 s of quiet, or at the latest 30 s after the first queued write, one commit per batch, by shelling out to the `git` CLI. The message is `memory: <tool> <path>` for one change (`memory: note_move <old path> → <new path>` for a move), and `memory: <n> changes` with one line per change for a batch. Author and committer come from `--git-author` (D13, default `memctl <memctl@localhost>`). It stages and commits only the paths a tool touched (`git commit --only`), so changes a person has staged stay as they are. A change made outside memctl, found by the freshness check or left uncommitted when memctl starts, is committed on its own as `memory: external change <path>`, with the repository's git identity when it has one (both `user.name` and `user.email`; a `WARN` at start names a missing key) and the `--git-author` identity otherwise, before any tool write to that path is committed; it is not folded into a tool's commit (D10), except an edit made to a note in the few seconds between a tool's write to it and that write's commit, which shares one commit with it; git hooks run as configured; commits are not signed, so a key that needs a passphrase or a touch can never block a background commit. It never pulls, pushes or touches other branches. If the root is not a git repo, commits are skipped and `memctl check` warns.

**Shutdown (D21).** On stdin EOF (the client closed), SIGINT or SIGTERM, `memctl mcp` stops taking tool calls, commits every queued path at once without waiting for the quiet window, and exits. One shutdown context reaches the MCP server and the commit goroutine.

## Deployment and config

One binary, `memctl`, with subcommands; config comes from flags and env vars only, no config file in v1.

| Command | Purpose |
| --- | --- |
| `memctl mcp --root ~/notes` | MCP server over stdio (D1) |
| `memctl version` | Print the version (SPEC-0004) |
| `memctl check` | Report parse errors, unsupported Markdown forms, dangling links, ambiguous links (a name several notes share), stale links (resolved only through an old path in `aliases`), observation or relation lines outside their sections, and a missing git repo |

Client setup is one line, e.g. `claude mcp add memory -- memctl mcp --root ~/notes`. Logs go to stderr as JSON (`log/slog`), never stdout, which carries the MCP protocol. Each tool call logs one `INFO` line after it ends: the tool, the duration, the error code if any, and counts (results, notes touched), never note content, queries or paths. Builds (D18): `make build` builds for the host; release builds with `CGO_ENABLED=0` for linux/amd64, linux/arm64 and darwin/arm64 via a GitHub Actions workflow that is started by hand (`workflow_dispatch`) for a given tag.

## Milestones

Three milestones, about 1.5–2 weeks in total with Claude Code, each ending in something usable; v2 adds vectors.

1. **M1 — Read-only memory.** Line parser, in-memory index, freshness check, `note_read`, `search`, `context`, `list`, `memctl check`. Exit: search and graph walk work over a copy of an existing notes root of a few hundred notes, and edits made in an editor show up on the next call.
2. **M2 — Writes.** Instance lock, `note_write`, `note_edit` with hash check, `note_move` with aliases, `note_delete`, atomic writes, git auto-commit. Exit: a Claude Code session creates, edits and renames notes, every change is one git commit, and a concurrent edit returns a conflict instead of a silent overwrite.
3. **M3 — Release.** Released only after the findings of the M1/M2 live check are fixed (stage plan 0004). A manually triggered release workflow in GitHub Actions (`workflow_dispatch`, input: an existing `vX.Y.Z` tag), README with client setup. Exit: a release binary runs end to end on a fresh notes root.
4. **v2 — Semantic search.** `Embedder` backed by TEI over HTTP, hybrid BM25 + cosine ranking.

## Backlog for 0.x

Small changes to v1 behaviour. Each needs a spec amendment before code.

None open. The three findings of the repeated live check of 2026-11-20
([`2026-11-20-live-check-repeat.md`](2026-11-20-live-check-repeat.md)) listed
here, N2 to N4, moved to SPEC-0002 and SPEC-0004 for v0.2.0 (see Amendments).

## Backlog after v2

None of these change v1 code; each is a later, independent addition.

- **HTTP mode:** streamable HTTP so several clients share one instance per root. Auth = a key generated on first start (32 random bytes in .memory/token, mode 0600), sent as a Bearer header and required even on loopback; memctl token show / rotate. Per-key git author via D13.
- **Per-type validation:** required frontmatter fields per `type` (e.g. `runbook`, `odr`), reported by `memctl check`.
- **MCP resources and prompts:** a session-start brief of recently changed notes.
- **Claude Code skill or hooks:** capture decisions into notes at session end.
- **Attachments:** index non-Markdown files (images, PDFs) by name and link; ignored in v1.

Vectors in v2 are chunked by heading and held in memory next to the BM25 index. Since embedding every note on each start would be slow, v2 adds one cache file, `.memory/vectors.bin`, keyed by each chunk's sha256; deleting it only costs a re-embed.

## Risks and open questions

The main risk is scale: the full in-memory index and the per-call freshness walk assume a notes root under about 10k files.

| Risk | Impact | Mitigation |
| --- | --- | --- |
| Notes root grows far beyond \~10k files | Slower start and tool calls | Measure in M1; if needed, add fsnotify or a persisted index as a later change |
| Editor and tool edit the same note | Lost human edit | `expected_sha` check; atomic rename; git history |
| BM25 ranking feels weak on short notes | Agents miss relevant notes | Tune field weights; title and alias boost; v2 vectors |
| Duplicate titles or aliases | Ambiguous title links | Paths are the unique key and the recommended link form (D23); an ambiguous link or `ref` is reported, never resolved by guess; `memctl check` lists ambiguous links |
| Unsupported Markdown forms (setext headings, HTML blocks) | Sections or links missed | `memctl check` flags them; notes use ATX headings |
| Two MCP clients configured on the same root (e.g. Claude Code and Claude Desktop) | Second client's server fails to start | Clear error with holder PID; one client per root (accepted for v1); HTTP sharing is a backlog item |

No open questions remain for v1.

## Follow-up after v1

An existing Basic Memory notes root moves to `memctl` once v1 is released, with no conversion step (lenient reads).

- [ ] `git init` in the notes root if it is not a git repo yet, and commit the current state as a baseline.
- [ ] Run `memctl check` on it and fix any observation or relation lines outside their sections by hand.
- [ ] Point the MCP client config at `memctl mcp --root <notes root>` and retire the Basic Memory server.

## Amendments

| Date | Decision | Change | Why |
| --- | --- | --- | --- |
| 2026-11-19 | D15 — Change detection | A file whose mtime is not at least 2 s older than the last read of its entry is hashed again even when mtime and size match ("racily clean"). | Two saves of the same size within one timestamp tick were missed until the next change (found by a property test in stage plan 0002, T12; SPEC-0003 R17). |
| 2026-11-19 | D24 — Move keeps old links working | `note_move` refuses a note without valid frontmatter, also for a path-only move. | The move could not record the old path in `aliases`, nor keep a title taken from the file name (SPEC-0007 R9). |
| 2026-11-20 | Search tokenizer | Identifiers joined by `-`, `.`, `_`, `/` or `:` are indexed whole as well as in parts; a query identifier matches only whole. | A search for `IT-104` or an IP address returned every note sharing one fragment (live check 2026-11-20, L3). |
| 2026-11-20 | D9 — `context` scope | The result lists every edge between the notes it returns, not only the edges the walk followed. | Edges between two notes at the maximum depth were missing, so the graph looked complete when it was not (live check, L4). |
| 2026-11-20 | Git — messages | One-change commits name the path, a move names the old and the new path. | Titles repeat across folders; `git log -- <path>` works on paths (live check, L11). |
| 2026-11-20 | Git — external changes | A change made outside memctl is committed on its own as `memory: external change <path>`, with the repository's git identity or the `--git-author` identity. | An editor change was committed inside memctl's next commit of that note, under memctl's name, against D10 (live check, L10). |
| 2026-11-20 | Milestones | M3 releases only after the live-check findings are fixed in stage plan 0004. | The maintainer chose to fix every finding before v0.1.0. |
| 2026-11-20 | Git — external changes | When the repository lacks `user.name` or `user.email`, a `WARN` at start names the missing key and the `--git-author` identity used instead. | A repository with a name but no email had a person's edits committed as memctl with no reason given (repeated live check, N1). |
| 2026-11-20 | Search snippets | A snippet of a line longer than 200 characters is a 200-character window centred on the first matched term, with `…` on each cut side. Which line is chosen does not change. | A snippet cut at the end of a long line could stop before the matched term, so the hit did not show why it matched (repeated live check, N2; SPEC-0002 R27). |
| 2026-11-20 | D9 — `context` scope | `context` takes `edges`: `all` (the default, as before) or `start`, which lists only the edges whose source or target is the start note; the notes are the same. The description says that depth 2 or 3 can return dozens of notes and hundreds of edges. | Depth 2 on a real root returned 33 notes and about 170 edges: complete, but costly in an agent's context (repeated live check, N3; SPEC-0002 R28, SPEC-0004 R31). |
| 2026-11-20 | Search tokenizer — suggestions | In `not_found` suggestions only, a query word of at least two letters also matches the same word followed by digits (`cisco` → `cisco1`). `search` results and ranking do not change. | Whole-word matching kept a near name such as `cisco1` out of the suggestions for `cisco` (repeated live check, N4; SPEC-0002 R29, SPEC-0004 R28). |
