# Project Guidelines

This file provides default AI agent instructions for this project. Read this file
first, then consult the relevant detail file before touching any code.

The product design, [`docs/design/solution-design.md`](docs/design/solution-design.md),
is the source of truth: when this file or any other doc disagrees with it, the
design wins. Behavioural specifications live under `docs/specs/` (what each
capability must do); architectural decisions are recorded under `docs/adr/` (why
each decision was made). Any non-trivial change follows the Spec → Plan →
Implement → Verify workflow in [`docs/agents/workflow.md`](docs/agents/workflow.md)
— read it before starting. Record notable, user-visible changes in
`CHANGELOG.md` as you make them (see
[`docs/agents/changelog.md`](docs/agents/changelog.md)).

---

## What this project is

`memctl` is a local-only MCP server in Go that gives AI clients a persistent
memory made of plain Markdown notes. It is one static binary, started by the MCP
client as a stdio child process (`memctl mcp --root <notes>`), with an in-memory
index (BM25F full-text search and a link graph) rebuilt from the files on every
start, a per-call freshness check that picks up edits made in any editor, and a
git commit after each tool write. `memctl check` reports problems in a notes
root. There is no HTTP transport, no authentication, no database and no config
file in v1.

The tools, the note format, the decisions D1–D24 and the milestones are in the
design. For the **current list of specs and their status**, see
`docs/specs/README.md` — that index is the single source of truth; do not
maintain a second copy of it here.

**Conventions to keep:** split by responsibility (ADR-0002); the Spec → Plan →
Implement → Verify workflow; test-first; one importer per third-party module
(ADR-0010), with go-sdk and yaml.v3 the only modules (ADR-0011); stdout carries
the MCP protocol only, logs go to stderr as JSON; tool results in Markdown with
a one-sentence first line and `<code>: <next step>` errors (ADR-0015); tool text
written for the agent with no internal references (ADR-0013); one log line per
tool call without note content (ADR-0016).

**Reserved for the maintainer:** never run `git push` or push tags. Commit
locally on `main` only.

---

## Detail Files

| Area | File |
|---|---|
| Product design (source of truth) | [`docs/design/solution-design.md`](docs/design/solution-design.md) |
| Development workflow (SDD) | [`docs/agents/workflow.md`](docs/agents/workflow.md) |
| Development environment (toolchain, build, local runs, agent permissions) | [`docs/agents/environment.md`](docs/agents/environment.md) |
| Go code conventions | [`docs/agents/backend.md`](docs/agents/backend.md) |
| MCP tool conventions (result shape, tool text, log line) | [`docs/agents/tools.md`](docs/agents/tools.md) |
| Writing ADRs (conventions) | [`docs/agents/adr.md`](docs/agents/adr.md) |
| ADR index (which decisions exist) | [`docs/adr/README.md`](docs/adr/README.md) |
| Keeping the changelog | [`docs/agents/changelog.md`](docs/agents/changelog.md) |
| Specifications (what the code must do) | [`docs/specs/README.md`](docs/specs/README.md) |

Always read the relevant detail file **before** making changes in that area. When
changing a capability that has a spec under `docs/specs/`, read that spec first
and keep it in sync with the change.

---

## Project Layout

```
memctl/
├── cmd/memctl/                 # entry point: subcommands mcp and check, flags, logger, shutdown
├── internal/
│   ├── mcpserver/              # MCP over stdio: tools, descriptions, result shape, tool-call log
│   ├── note/                   # note service: CRUD, freshness check, instance lock, atomic writes
│   ├── parse/                  # line-based Markdown parser + yaml.v3 frontmatter
│   ├── index/                  # in-memory index: BM25F, names, backlinks, graph walk
│   ├── gitlog/                 # git committer: queue, quiet window, external changes, final flush
│   ├── share/                  # shared instance: owner socket, forwarder handshake, byte pipe
│   └── arch/                   # module-wide tests: one importer per third-party module (ADR-0010)
├── scripts/                    # check-docs.sh, release-notes.sh, release-build.sh and their tests
├── .github/workflows/          # release.yml: the manual release workflow (ADR-0012)
├── docs/
│   ├── design/                 # the product design — source of truth
│   ├── adr/                    # architecture decision records; README.md is the index
│   ├── specs/                  # capability specs and their plans
│   ├── plans/                  # stage plans (ADR-0018)
│   └── agents/                 # working guides (the how)
├── Makefile                    # build, test, verify (make help)
├── .mise.toml                  # Go pin (ADR-0012)
├── .claude/settings.json       # committed agent tool permissions
├── CLAUDE.md                   # this file
├── AGENTS.md                   # pointer to CLAUDE.md
├── CHANGELOG.md                # human-readable release notes
├── README.md                   # user-facing docs
└── LICENSE                     # MIT
```

When adding new functionality, follow ADR-0002 (split by responsibility) — add a
new file or package named after what it does rather than appending to an
unrelated existing file.
