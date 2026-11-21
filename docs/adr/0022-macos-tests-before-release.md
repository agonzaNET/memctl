# ADR-0022 - Run the test suite on macOS before a release is built (amends ADR-0012)

## Status

Proposed

**Area:** Build

## Context

ADR-0012 builds release binaries for linux/amd64, linux/arm64 and
darwin/arm64, and the release workflow runs `make verify` on an Ubuntu
runner only. The development VM is Linux too. So the darwin binary, the one
the maintainer runs in Claude Desktop, is cross-built but no test has ever
run on macOS.

Until v0.2.0 that gap was small: the code used little that differs between
the two systems. v0.3.0 (ADR-0021, SPEC-0009) adds a Unix socket whose path
limit is 103 bytes on macOS against 107 on Linux, relies on `flock` across
processes, on signal handling and on SIGPIPE behaviour for closed pipes.
The release audit of 2026-11-21 listed this as warning W6: macOS behaviour
is first seen in a manual live check.

GitHub offers macOS runners on Apple silicon. The suite needs Go and `git`,
both available there; tests that need Linux (`/proc`) skip themselves.

## Decision

> We will run `go vet ./...` and `go test ./...` on a macOS runner, with the
> Go version pinned in `.mise.toml`, in a workflow that can be started by
> hand for any ref and that the release workflow calls for the tag. The
> release job needs it, so no release is built when a test fails on macOS.

- A new workflow, `.github/workflows/test-macos.yml`, with two triggers:
  `workflow_dispatch` (input: the ref, default `main`) to run it before
  tagging, and `workflow_call` (input: the ref) for the release workflow.
- Runner: `macos-15` (Apple silicon, the architecture of the darwin binary).
  Same pinned actions and the same Go version step as `release.yml`.
  `permissions: contents: read`; no secrets.
- `release.yml` gets a job `test-macos` that calls the new workflow with the
  tag, and the `release` job `needs: test-macos`. Linux keeps `make verify`
  and `govulncheck` in the release job as before (ADR-0020).
- Only Go checks run on macOS, not all of `make verify`: the docs check and
  the script tests do not depend on the platform and already run on Linux.
- Nothing runs on push or tag; the "started by hand" rule of ADR-0012 stays.

## Alternatives Considered

| Alternative | Reason not chosen |
|---|---|
| Rely on the maintainer's live check on macOS | Manual, done only when a release changes behaviour the maintainer checks by hand; a regression in an untouched area would ship |
| A macOS job inside `release.yml` only | A macOS failure is found only after the tag is pushed, so fixing it means a new patch tag or moving a published tag; the separate workflow can run on `main` first |
| `make verify` on macOS | Repeats the platform-independent docs and script checks, and pulls macOS's bash 3.2 and BSD tools into the gate for no gain |
| A test matrix that runs on every push | Against ADR-0012's "nothing builds on push or tag"; costs runner time for every docs commit |
| Running the tests on the maintainer's Mac | Development does not happen on the Mac (the code is written in a Linux VM); depends on one machine and on memory, as govulncheck did before ADR-0020 |

## Consequences

- **Positive:** Every release has passed the full Go test suite on the
  operating system and architecture of the darwin binary.
- **Positive:** The maintainer can run the macOS tests on `main` before
  tagging, with one command:
  `gh workflow run test-macos.yml -f ref=main`.
- **Negative:** A release takes a few minutes longer; on a private repository,
  macOS runner minutes count ten times against the quota (free on a public
  one).
- **Risk / Note:** The first run may show tests that assume Linux (paths,
  error texts). Each one is a finding, fixed in the test or the code before
  the release, never by skipping it on macOS without a reason in the test.

## References

- [ADR-0012](0012-build-targets-and-development-environment.md) — build targets and the release workflow.
- [ADR-0020](0020-vulnerability-check-in-release-workflow.md) — govulncheck in the release workflow.
- [ADR-0021](0021-shared-instance-over-unix-socket.md) — the shared instance.
- Release audit of the v0.3.0 candidate, warning W6:
  [`../design/2026-11-21-v0.3.0-release-audit.md`](../design/2026-11-21-v0.3.0-release-audit.md).
- GitHub Actions: reusing workflows (`workflow_call`), macOS runners.
