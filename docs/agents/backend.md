# backend.md — Go code

Read this before touching any code in this repository. Also read
[`tools.md`](tools.md) before touching the MCP tools, and [`adr.md`](adr.md)
before proposing any architectural change. The product itself — note format,
index, tools, git behaviour — is defined in
[`../design/solution-design.md`](../design/solution-design.md).

## ADRs governing this area

Read the **foundational ADRs** that apply to every change (see
[`adr.md`](adr.md#foundational-adrs--always-in-force)) plus the area-local ADRs
in Areas **Build**, **Tools** and **Lifecycle**. Find them in the
[ADR index](../adr/README.md#index) by filtering on those Areas.

---

## Stack

- **Language:** Go. The `go` directive in `go.mod` is the minimum, and
  `.mise.toml` pins the toolchain (ADR-0012, [`environment.md`](environment.md)):
  when the directive rises, raise the pin in the same commit.
- **Dependency policy:** stdlib-first. Two third-party modules are allowed
  (ADR-0011): `github.com/modelcontextprotocol/go-sdk`, imported only by
  `internal/mcpserver`, and `go.yaml.in/yaml/v3`, imported only by
  `internal/parse`. Any other module needs an ADR first.
  `internal/arch/imports_test.go` fails on a second importer or an unlisted
  module (ADR-0010); add a module to its map in the change that first imports
  it.
- **No cgo.** Every build is `CGO_ENABLED=0`.
- **git** is used through the `git` CLI (`os/exec`), never a Go git library.

---

## Package layout

Follow ADR-0002: split by responsibility, not line count. One package per
nameable concern. The layout from the design (D20):

| Package / path | Single responsibility |
|---|---|
| `cmd/memctl/` | Entry point: subcommands (`mcp`, `check`), flags, logger, shutdown context, wiring |
| `internal/mcpserver/` | The MCP server over stdio: tool registration, descriptions, schemas, result and error shape, the tool-call log line. The only importer of the MCP SDK |
| `internal/note/` | The note service: create, read, edit, move, delete; freshness check; instance lock; atomic writes |
| `internal/parse/` | The line-based Markdown parser and frontmatter (yaml.v3). The only importer of yaml.v3 |
| `internal/index/` | The in-memory index: notes, names, backlinks, inverted index, BM25F, graph walk |
| `internal/gitlog/` | The git committer: queue, quiet window, final flush, `git` CLI calls |
| `internal/share/` | The shared instance: the owner's Unix socket, the forwarder handshake and the byte pipe between them (SPEC-0009, ADR-0021) |
| `internal/arch/` | No code: module-wide tests, such as one importer per third-party module (ADR-0010) |

A new concern gets a new package, not a new file in a package that already has
a different responsibility. A package owns its types and the narrow interface
its callers depend on, so callers can stub it in tests.

---

## Adding a capability

Follow the workflow in [`workflow.md`](workflow.md) (Spec → Plan → Implement →
Verify). For a new or changed tool, also follow the checklist in
[`tools.md`](tools.md#adding-or-changing-a-tool).

- **Domain first.** Put the logic in the package that owns the concept
  (`note`, `index`, `parse`, `gitlog`); the MCP layer only parses arguments,
  calls the note service and formats the result.
- **Test-first.** Write table-driven tests beside the package under test, named
  after the spec requirement they prove, *before* the implementation (ADR-0005).
- **Verify.** `make verify` clean; update `CHANGELOG.md`.

---

## Cross-cutting concerns

### Configuration

- Configuration comes from flags and environment variables only; there is no
  config file in v1 (design, "Deployment and config"). A new flag gets a line in
  the spec that adds it, in the startup log line, and in `README.md`.
- An invalid value stops the start with an error that names the flag and the
  rule, never the value.
- Config is parsed once in `cmd/memctl`; packages receive plain Go values, never
  a global config object.

### Logging

- Use `log/slog` with the JSON handler on **stderr**. stdout carries the MCP
  protocol; nothing else may ever write to it, including `fmt.Print` and
  library output.
- Initialise the logger once in `cmd/memctl` and pass it to packages that need
  it.
- Use key-value attributes, never `fmt.Sprintf` in a message.
- Levels: `ERROR` for unexpected failures, `WARN` for recoverable conditions,
  `INFO` for lifecycle events (start, index built, shutdown) and the one
  `tool call` line per tool call (ADR-0016), `DEBUG` for detail.
- Never log note content, frontmatter values, queries, titles, aliases or
  paths (ADR-0016).

### Error handling

- Wrap errors with context: `fmt.Errorf("parse %s: %w", rel, err)` inside the
  domain packages.
- Return errors up to the MCP layer; do not log and return — pick one.
- Define sentinel errors (for example `note.ErrExists`, `note.ErrConflict`) in
  the package that owns the concept; check them with `errors.Is`/`errors.As`.
- The MCP layer maps each sentinel to one error code of the fixed list and a
  sentence that says what to do next (ADR-0015). An unexpected error maps to
  `internal_error` with a fixed sentence and is logged once.
- Never swallow errors silently.

### Files and concurrency

- Files always win: a write never overwrites a file in a state the tool has not
  seen (`expected_sha`), and every write is temp file, `fsync`, `rename` in the
  same directory.
- Tool calls are serialised by one mutex in the note service.
- Paths from tool arguments are relative to the notes root, cleaned, and must
  stay inside it; `..`, absolute paths and symlinks that leave the root are
  refused.

---

## Testing

- **Framework:** stdlib `testing` only — no testify, no assertion libraries.
- **Style:** table-driven tests. Each sub-test name states the scenario; the
  test name states the spec requirement it proves (e.g.
  `TestParse_SPEC0001_R4_headingInFenceIgnored`).
- **Fixtures:** notes roots under `testdata/` or built in `t.TempDir()`; git
  tests run `git init` in a temp dir and skip when `git` is missing.
- **MCP:** test tools through an in-memory client–server pair of the SDK, not by
  spawning the binary; one end-to-end test may spawn `memctl mcp` over pipes.
- **Commands:** `go test ./...`, `go vet ./... && gofmt -l .`, `make verify`.
- **Coverage target:** every spec requirement (`R1`, `R2`, …) has at least one
  passing test that names it. Uncovered requirements are unfinished work.
- Test files live beside the package they test; use the `_test` package suffix
  for black-box tests.
