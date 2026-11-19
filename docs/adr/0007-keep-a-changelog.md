# ADR-0007 - Keep a human-readable changelog

## Status

Accepted

<!--
Status values: Proposed | Accepted | Rejected | Deprecated | Superseded by ADR-NNNN
-->

**Area:** Methodology

## Context

ADRs record *why* decisions were made, and the commit history (ADR-0006) records
*what changed per task* in fine-grained, spec-linked detail. Neither gives a
consumer or operator a concise, curated answer to "what changed in this release?"
Commit logs are too granular and noisy for that audience — they mix internal task
commits, refactors, and red/green TDD steps with the handful of changes an outside
reader actually cares about.

A curated, human-readable changelog fills that gap and, paired with Semantic
Versioning, communicates the *impact* of a release (breaking vs. feature vs. fix)
at a glance.

## Decision

> We will maintain a `CHANGELOG.md` at the repository root, following the *Keep a
> Changelog* format and Semantic Versioning.

- The file lives at the repo root and is reverse-chronological (newest first).
- A single `[Unreleased]` section sits at the top and accumulates changes as they
  land; on release it is renamed to the version and date, and a fresh
  `[Unreleased]` is opened above it.
- Entries are grouped under the standard headings: **Added, Changed, Deprecated,
  Removed, Fixed, Security.**
- Entries are written for humans (consumers/operators), describing the observable
  effect — not copied from commit messages.
- The changelog is updated **in the same change** as the behaviour it describes,
  mirroring the spec-sync rule from ADR-0003. This is Phase 4 of the workflow.
- Versions follow Semantic Versioning (`MAJOR.MINOR.PATCH`): breaking changes bump
  MAJOR, backwards-compatible features bump MINOR, backwards-compatible fixes bump
  PATCH. While the MAJOR version is `0`, a breaking change bumps MINOR instead
  ([ADR-0008](0008-pre-1-0-versioning.md)).
- An entry may reference its governing spec (`SPEC-NNNN`) where that helps a
  reader, consistent with ADR-0006.

The mechanics (categories, entry style, release flow) live in
`docs/agents/changelog.md`; this ADR records the decision and its boundaries.

## Alternatives Considered

| Alternative | Reason not chosen |
|---|---|
| Rely on git log / commit history alone | Too granular and noisy for consumers; mixes internal task commits with user-facing changes; no SemVer grouping or curated summary. |
| Auto-generate the changelog from Conventional Commits | Tempting given ADR-0006, but produces a commit dump rather than a curated, human-readable summary. Could be revisited as a tooling decision later; the curated file is the contract. |
| Keep release notes only in the hosting platform (e.g. GitHub Releases) | Ties the record to a platform and keeps it out of the version-controlled repo where the code and its history live. |
| No changelog; point readers at tags/diffs | Leaves readers to reconstruct what changed from raw diffs — exactly the work a changelog exists to save. |

## Consequences

- **Positive:** Consumers and operators get one curated, in-repo, version-controlled
  view of notable changes per release.
- **Positive:** Paired with SemVer, the changelog communicates the impact of each
  release at a glance.
- **Negative:** Another artifact to keep in sync; a stale changelog misleads.
  Mitigated by updating it in the same change (workflow Phase 4) and by keeping
  entries short.
- **Note:** The *how* (format, categories, release flow) lives in
  `docs/agents/changelog.md`.

## References

- ADR-0003 — Adopt Spec-Driven Development (the "update in the same change" guardrail)
- ADR-0006 — Conventional commit messages referencing the spec (per-task history; the changelog is its curated, per-release counterpart)
- `docs/agents/changelog.md` — the how-to guide
- Keep a Changelog — https://keepachangelog.com
- Semantic Versioning — https://semver.org
