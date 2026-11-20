# changelog.md — Keeping the changelog

How the project's `CHANGELOG.md` is maintained. The decision to keep one — and
why — is ADR-0007; this guide is the *how*. The changelog is the curated,
human-readable record of notable changes per release, aimed at consumers and
operators. It is the per-release counterpart to the spec-linked commit history
(ADR-0006): commits are the fine-grained trail, the changelog is the summary.

`CHANGELOG.md` lives at the repository root.

---

## Format (Keep a Changelog)

- Newest first. The top section is always `## [Unreleased]`.
- Each released version is `## [X.Y.Z] - YYYY-MM-DD`.
- Within a section, group entries under the headings that apply — omit the ones
  with no entries.
- Keep entries short: one change per entry, a few lines at most, written for a
  reader of the project, not for its authors. Say what the reader sees and, for
  an operator, what to do.

### No non-standard sections

`[Unreleased]` has no **Pending work** section. The gap between the documented
target and the shipped code lives in [`../specs/README.md`](../specs/README.md)
(the spec statuses), not here. Do not add a non-standard heading for it or for
anything else.

## Categories

Use the standard set; each entry goes under exactly one:

| Heading | Use for |
|---|---|
| **Added** | New features or capabilities. |
| **Changed** | Changes to existing behaviour. |
| **Deprecated** | Soon-to-be-removed features (still present, flagged). |
| **Removed** | Features removed in this release. |
| **Fixed** | Bug fixes. |
| **Security** | Vulnerability fixes or hardening. |

## When to add an entry

- Add an entry under `[Unreleased]` for any change a consumer or operator can
  observe: behaviour, a public interface or contract, configuration, defaults, or
  operational steps.
- Add it **in the same change** as the code (same commit / PR), mirroring the
  spec-sync rule in ADR-0003 — a changelog updated "later" drifts.
- Skip purely internal changes nobody outside the code can observe (a refactor
  with identical behaviour, a test-only change). When unsure whether a change is
  observable, add the entry.

## Writing good entries

- Describe the **effect**, not the implementation. "Reject requests with no auth
  token (401)" — not "added `checkToken()` to middleware".
- One change per entry; lead with the user-visible behaviour.
- Reference the governing spec where it helps: `… (spec: SPEC-NNNN)`.
- Call out breaking changes explicitly — prefix with `**BREAKING:**` — because
  they force a MAJOR bump at the next release (a MINOR bump while the version
  is `0.y.z`, ADR-0008).
- Do not paste commit messages or a `git log` dump.

## Releasing

When cutting a release:

1. Rename `## [Unreleased]` to `## [X.Y.Z] - YYYY-MM-DD` (today's date).
2. Open a fresh, empty `## [Unreleased]` above it, and move the HTML
   maintenance comment up under it. (`scripts/release-notes.sh` drops comment
   lines from the release notes anyway.)
3. Choose `X.Y.Z` by Semantic Versioning against the entries being released:
   - **MAJOR** — any breaking change.
   - **MINOR** — backwards-compatible new behaviour (an `Added`, or a compatible
     `Changed`).
   - **PATCH** — backwards-compatible bug fixes only.

   **Before `1.0.0`** (ADR-0008): a breaking change bumps **MINOR**, not
   MAJOR, so `0.1.0` plus a breaking change is `0.2.0`, never `0.1.1` or
   `1.0.0`. The other two rules stay the same. Going to `1.0.0` is a separate
   decision that the contract is stable.
4. Optionally maintain version-comparison links at the bottom of the file
   (replace the `[Unreleased]` link to `commits/main` with a compare link and
   add one for the new tag).
5. Update the version in the status note at the top of `README.md`, if it
   has one.
6. Run `make verify`, commit, then tag that commit `vX.Y.Z`. The Makefile
   strips the `v`, so a build from the tag reports `X.Y.Z` as the version
   in the MCP server info. The maintainer pushes the
   commit and the tag, then starts the release workflow by hand in GitHub
   Actions with that tag as input; it builds the binaries and attaches them to
   the release (ADR-0012).

Example shape:

```markdown
## [Unreleased]

## [1.2.0] - 2025-06-01

### Added
- Token-based authentication for the admin API (spec: SPEC-0006).

### Fixed
- Config loader no longer panics on a missing optional field.
```

## Rules for AI agents

- Update `[Unreleased]` as part of Phase 4 of the workflow (Verify & commit), in
  the same commit as the change — see [`workflow.md`](workflow.md).
- Never invent a version number or a date for `[Unreleased]` entries; they get a
  version and date only at release time.
- Never edit an already-released section to change what it says — a released
  changelog entry is a historical record (same spirit as not editing an Accepted
  ADR).
- Categorise correctly and keep entries consumer-facing.

## Anti-patterns

- Dumping commit messages or `git log` output into the changelog.
- Leaving `[Unreleased]` empty after shipping a user-visible change.
- Inventing a release date, or dating `[Unreleased]`.
- Retroactively rewriting a released section.
- Burying a breaking change in `Changed` without flagging it.
