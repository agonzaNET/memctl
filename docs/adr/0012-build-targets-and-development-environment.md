# ADR-0012 - Build for the host locally, build release binaries for three platforms in CI, and pin the toolchain with mise

## Status

Accepted

**Area:** Build

## Context

`memctl` is started by its MCP client as a child process (D1), so it runs on
the machine where the client runs: mostly macOS on Apple silicon, and Linux on
amd64 or arm64. There is no deployment host and no service unit.

The build must give one static binary per platform with no runtime
dependencies. Go cross-compiles pure-Go code cleanly when cgo is off. A
developer also needs a binary that runs on the machine they are working on,
for example to point Claude Code at it.

Development can happen on a laptop or in a Linux VM. The code shells out to
`git` and uses `flock`, which behave the same on Linux and macOS for our use.

## Decision

> `make build` builds a binary for the host with `CGO_ENABLED=0`. A manually
> triggered GitHub Actions workflow builds the release binaries for `linux/amd64`, `linux/arm64` and
> `darwin/arm64`. `mise` pins the Go toolchain; a dedicated Linux VM is
> recommended for development but not required.

- **Local build.** `make build` writes `bin/memctl` for the host's `GOOS` and
  `GOARCH`, with the version from the latest `vX.Y.Z` tag injected by
  `-ldflags`. `GOOS`/`GOARCH` may be overridden for a one-off build.
- **Release build.** A GitHub Actions workflow builds the three targets and
  attaches the binaries to a GitHub release (milestone M3). For now it runs
  only when started by hand (`workflow_dispatch`), with an existing `vX.Y.Z`
  tag as its input; it has no `push`, `tag` or `schedule` trigger. It builds
  from that tag, never pushes code and never creates or moves tags. An
  automatic trigger on tag push is a later, separate decision.
- **Toolchain.** `.mise.toml` pins Go. The `go` directive in `go.mod` is the
  minimum; when it rises, the pin rises in the same commit.
- **Dev machine.** An isolated Linux VM keeps an agent's tool permissions
  (`.claude/settings.json`) away from personal files and is the recommended
  setup. A laptop works too. Keep the checkout on the VM's own disk, not on a
  host share, so that a host git and a VM git never write the same index.
- **Platforms not built.** Windows is not a target in v1 (`flock` and the
  stdio spawn model are not tested there).

## Alternatives Considered

| Alternative | Reason not chosen |
|---|---|
| `make build` always cross-compiles to one fixed target | The binary would not run on the developer's own machine, where the MCP client is. |
| Build releases automatically on every tag push | The release process is still new; a manual start keeps each release a deliberate step until it has run a few times. |
| Build all release targets locally | Slower local loop; the release artifacts should come from one clean, repeatable place. |
| Require a dev VM | Adds setup for a single-user tool whose builds and tests run the same on macOS. |
| Homebrew or system Go instead of mise | Not pinned per repository; versions drift between machines. |

## Consequences

- **Positive:** The local binary runs where it is built; CI produces the
  three release binaries the same way every time.
- **Positive:** The tests run on the developer's platform, which is one of the
  release platforms.
- **Negative:** A platform-specific bug on a release target the developer
  does not use only shows up in CI or in use.

## References

- Design D1, D7, D18 in [`../design/solution-design.md`](../design/solution-design.md).
- [`../agents/environment.md`](../agents/environment.md).
