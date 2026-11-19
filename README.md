# memctl

Local Markdown memory for AI agents: a single-binary Go MCP server with
full-text search, a link graph and git-backed history.

**Status:** design settled, implementation not started. See
[`docs/design/solution-design.md`](docs/design/solution-design.md) for what is
being built, and [`docs/specs/README.md`](docs/specs/README.md) for progress.

## What it does

- Your notes stay plain Markdown files in one folder; edit them in any editor.
- `memctl mcp --root ~/notes` serves them to an MCP client over stdio: create,
  read, edit, move and delete notes, full-text search, and a walk over the links
  between notes.
- Every change made by a tool is a git commit in the notes folder, so it can be
  reviewed and undone. `memctl` never pushes.
- `memctl check --root ~/notes` reports parse problems, dangling links and
  duplicate titles.

Planned client setup, once released:

```sh
claude mcp add memory -- memctl mcp --root ~/notes
```

## Building

```sh
mise install    # the pinned Go toolchain
make build      # bin/memctl for this machine
make verify     # tests, vet, gofmt and the docs checks
```

See [`docs/agents/environment.md`](docs/agents/environment.md).

## Design and decisions

- [`docs/design/`](docs/design/) — the product design (the source of truth).
- [`docs/adr/`](docs/adr/) — architecture decision records (the why).
- [`docs/specs/`](docs/specs/) — capability specs (the what).
- [`CHANGELOG.md`](CHANGELOG.md) — notable changes per release.

## License

[MIT](LICENSE)
