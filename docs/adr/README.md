# Architecture Decision Records — Index

This directory holds the project's **Architecture Decision Records** (ADRs): the
record of *why* each significant technical decision was made. One file per
decision, `docs/adr/<four-digit-index>-<short-slug>.md`; the index is sequential
and never reused, even when an ADR is superseded (ADR-0001).

This file is the **registry** — the authoritative list of which ADRs exist and
their status (ADR-0009). The *conventions* for ADRs (when to write one, status
and supersession rules, the rules for AI agents, and the controlled `Area`
vocabulary) live in the agent guide [`../agents/adr.md`](../agents/adr.md); the
*format* of an ADR lives in [`TEMPLATE.md`](TEMPLATE.md).

## How ADRs relate to the other docs

| Doc type | Answers | Lives in |
|---|---|---|
| **Design** | What is the product, and why, as a whole? | `docs/design/` |
| **ADR** (here) | *Why* was a technical decision made? What were the alternatives? | `docs/adr/` |
| **Spec** | *What* must the system do? Inputs, outputs, acceptance criteria. | `docs/specs/` |
| **Agent guide** | *How* do I work in this area of the code? | `docs/agents/` |

The design under `docs/design/` is the source of truth; where an ADR and the
design disagree, the design wins (see [`../design/README.md`](../design/README.md)).

## Foundational ADRs — always in force

These decisions are **global**: they apply to every non-trivial change regardless
of area.

- [ADR-0002](0002-code-modularity.md) — Code modularity (split by responsibility).
- [ADR-0003](0003-adopt-spec-driven-development.md) — Spec-Driven Development.
- [ADR-0004](0004-spec-plan-implement-verify-workflow.md) — Spec → Plan → Implement → Verify ([ADR-0018](0018-specs-first-and-stage-plans.md): specs first, stage plans).
- [ADR-0005](0005-test-first-development.md) — Test-first development.
- [ADR-0006](0006-conventional-commits-referencing-specs.md) — Conventional commits referencing the spec.
- [ADR-0007](0007-keep-a-changelog.md) — Keep a human-readable changelog ([ADR-0008](0008-pre-1-0-versioning.md) for pre-1.0 versions).

Every other ADR is **area-local**: find the ones relevant to your change by its
**Area** in the table below.

## Maintaining this index

Every ADR carries an `**Area:**` line (under its Status) drawn from the controlled
vocabulary documented in
[`../agents/adr.md`](../agents/adr.md#area-vocabulary) — currently
`Methodology`, `Build`, `Tools`, `Lifecycle`, `Index`.

Update the table every time a new ADR is added or an ADR's status or area changes.
`scripts/check-docs.sh` fails if a row's **Status** or **Area** disagrees with the
ADR file it points to. When an ADR becomes `Superseded by ADR-NNNN` or
`Deprecated`, move its row from **Active** to **Superseded / Deprecated** below
(keeping the file and its ID).

## Index

### Active

| ID | Title | Status | Area |
|---|---|---|---|
| [ADR-0001](0001-record-architecture-decisions.md) | Record architecture decisions | Accepted | Methodology |
| [ADR-0002](0002-code-modularity.md) | Code modularity | Accepted | Methodology |
| [ADR-0003](0003-adopt-spec-driven-development.md) | Adopt Spec-Driven Development with specs under docs/specs | Accepted | Methodology |
| [ADR-0004](0004-spec-plan-implement-verify-workflow.md) | Adopt the Spec → Plan → Implement → Verify workflow | Accepted | Methodology |
| [ADR-0005](0005-test-first-development.md) | Test-first development for non-trivial logic | Accepted | Methodology |
| [ADR-0006](0006-conventional-commits-referencing-specs.md) | Conventional commit messages referencing the spec | Accepted | Methodology |
| [ADR-0007](0007-keep-a-changelog.md) | Keep a human-readable changelog | Accepted | Methodology |
| [ADR-0008](0008-pre-1-0-versioning.md) | Bump MINOR, not MAJOR, for a breaking change before 1.0.0 | Accepted | Methodology |
| [ADR-0009](0009-adr-index-with-areas.md) | Keep one ADR index next to the ADRs, with an area per ADR and a list of foundational ADRs | Accepted | Methodology |
| [ADR-0010](0010-one-importer-per-third-party-module.md) | Confine each third-party module to one package, and enforce it with an import-scan test | Accepted | Methodology |
| [ADR-0011](0011-mcp-sdk-and-yaml-v3.md) | Use the official Go MCP SDK and yaml.v3 as the only third-party modules | Accepted | Build |
| [ADR-0012](0012-build-targets-and-development-environment.md) | Build for the host locally, build release binaries for three platforms in CI, and pin the toolchain with mise | Accepted | Build |
| [ADR-0013](0013-agent-first-tool-text.md) | Write tool descriptions and schemas for the agent, and check them for internal references | Accepted | Tools |
| [ADR-0014](0014-shutdown-flushes-pending-commits.md) | Stop on stdin EOF or a signal with one shutdown context, and commit every queued path before exit | Accepted | Lifecycle |
| [ADR-0015](0015-tool-result-shape.md) | Return Markdown tool results with a one-sentence first line, and errors as a fixed code plus a next step | Accepted | Tools |
| [ADR-0016](0016-tool-call-log-without-content.md) | Log one line per tool call, without note content | Accepted | Tools |
| [ADR-0017](0017-path-identity-and-link-resolution.md) | Identify notes by path, resolve links path first, report ambiguity, and keep old paths as aliases on a move | Accepted | Index |
| [ADR-0018](0018-specs-first-and-stage-plans.md) | Write the specs of a stage before any code, and plan a stage of coupled specs in one plan (amends ADR-0004) | Accepted | Methodology |
| [ADR-0019](0019-go-yaml-in-module.md) | Take YAML support from go.yaml.in/yaml/v3, the maintained continuation of yaml.v3 (amends ADR-0011) | Accepted | Build |
| [ADR-0020](0020-vulnerability-check-in-release-workflow.md) | Run govulncheck in the release workflow and stop the release on a reachable vulnerability (amends ADR-0012) | Accepted | Build |
| [ADR-0021](0021-shared-instance-over-unix-socket.md) | Share one memctl process per notes root: the first process owns it, later ones forward over a Unix socket (amends ADR-0014) | Proposed | Lifecycle |

### Superseded / Deprecated

| ID | Title | Status | Area |
|---|---|---|---|

None yet.

Where a later decision **extends or amends** an earlier one rather than replacing
it, the earlier ADR stays `Accepted` and carries a `> **Amended by ADR-NNNN**`
pointer note at its top, and the later ADR names the amendment in its title
(`… (amends ADR-NNNN)`).
