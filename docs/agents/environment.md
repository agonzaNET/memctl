# environment.md — Development environment

Where this project is developed and how to get a working toolchain. Read this
before running build, test or install commands. The *why* is
[ADR-0012](../adr/0012-build-targets-and-development-environment.md); this file
is the *how*.

`memctl` has no deployment host: the MCP client starts it as a child process on
the user's machine (design D1). The machine that builds it and the machine that
runs it are often the same.

---

## ADRs governing this area

Read the **foundational ADRs** (see
[`adr.md`](adr.md#foundational-adrs--always-in-force)) plus the area-local ADRs
in Area **Build** in the [ADR index](../adr/README.md#index).

---

## Where to develop

A dedicated Linux VM (for example one managed by Lima or Colima) is the
recommended setup: it keeps an agent's tool permissions away from personal
files. A laptop works too; the build and the tests are the same on macOS and
Linux.

> **Keep the tree off a host share.** A checkout on a share from the host (a
> virtiofs mount under `/Users/…`) shares `.git/index` with any git that runs
> on the host, for example a desktop app that polls `git status`, and two gits
> writing one index can record the deletion of most of the repository. If the
> tree must stay on a share, check `git show --stat` after each commit.

## Bootstrapping the toolchain

`mise` owns the language toolchain; the OS package manager owns the rest.

**1. Supporting packages** (once per machine): `make`, `git`, `curl`, `jq`.
On Ubuntu:

```bash
sudo apt-get update && sudo apt-get install -y make git curl jq
```

**2. `mise`** (once per machine), then activate it in your shell:

```bash
curl -fsSL https://mise.run | sh && echo 'eval "$(~/.local/bin/mise activate bash)"' >> ~/.bashrc
```

**3. The pinned toolchain** — from the repo root, reading [`.mise.toml`](../../.mise.toml):

```bash
mise install
go version
```

When the `go` directive in `go.mod` rises, raise the pin in `.mise.toml` to at
least it in the same commit. Nothing enforces that they agree.

## The development loop

```bash
make build      # bin/memctl for this machine (CGO_ENABLED=0, version from the latest vX.Y.Z tag)
make test       # go test ./...
make verify     # test + vet + fmt-check + check-docs — the gate of workflow Phase 4
```

Run [`scripts/check-docs.test.sh`](../../scripts/check-docs.test.sh) as well
whenever you edit `check-docs.sh` itself; `make verify` does not.

For a one-off binary for another platform:
`make build GOOS=linux GOARCH=amd64`. Release binaries for `linux/amd64`,
`linux/arm64` and `darwin/arm64` are built by the GitHub Actions release
workflow (milestone M3), which the maintainer starts by hand for an existing
`vX.Y.Z` tag; nothing builds on push or tag.

## Running it locally

Point an MCP client at the local binary and a scratch notes root, never at
your real notes while developing:

```bash
mkdir -p /tmp/memctl-notes && git -C /tmp/memctl-notes init
claude mcp add memctl-dev -- "$PWD/bin/memctl" mcp --root /tmp/memctl-notes
```

`memctl check --root /tmp/memctl-notes` runs without the instance lock and
without a client.

The tests never need a client, a network or real notes: they build notes roots
in temp directories, and drive the tools through an in-memory MCP client.

## Agent tool permissions

Agent permissions are committed in `.claude/settings.json`, so the trust
boundary is reviewable rather than re-decided each session:

- **Pre-approved:** the read-and-verify loop — `go build`/`test`/`vet`/`fmt`,
  `make build`/`test`/`verify`, `scripts/check-docs.sh` and its test,
  `mise install` and the read-only `mise` commands, read-only `git`, local
  commits (`git add`, `git commit`), and ordinary file inspection.
- **Still asks:** anything outward-facing or host-mutating — `sudo`,
  `git push`, `git tag`, `gh`, package installation, `curl`, and a `find` that
  deletes, runs a command or writes a file.
- **Denied:** recursive deletes outside the repo, history rewrites, and reading
  secret material (`*.pem`, `*.key`, `.env`, `*.env`, `~/.ssh`).

Adjust the lists there rather than disabling permissions wholesale.
