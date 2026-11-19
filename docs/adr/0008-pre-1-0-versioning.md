# ADR-0008 - Bump MINOR, not MAJOR, for a breaking change before 1.0.0

## Status

Accepted

**Area:** Methodology

## Context

[ADR-0007](0007-keep-a-changelog.md) says versions follow Semantic Versioning
and that breaking changes bump MAJOR. It does not say what happens while the
MAJOR version is still `0`. `memctl` starts at `0.1.0`, and its tool surface
and note format will still move after the first release.

Read word for word, ADR-0007 would make the first breaking change after
`v0.1.0` release `v1.0.0`, a version that claims a stable contract. A PATCH
release is wrong for a different reason: PATCH means backwards-compatible bug
fixes only.

Semantic Versioning item 4 already covers this case: major version zero is for
initial development, and anything may change. The common practice under
`0.y.z` is to bump MINOR for a breaking change.

## Decision

> While the MAJOR version is `0`, we will bump **MINOR**, not MAJOR, for a
> breaking change.

The rest of ADR-0007's rule does not change: a backwards-compatible feature
bumps MINOR, and a backwards-compatible fix bumps PATCH. The changelog still
prefixes every breaking entry with `**BREAKING:**`, so a reader of a `0.y.0`
release sees what breaks.

From `1.0.0` on, ADR-0007's rule applies unchanged. Moving to `1.0.0` is a
deliberate decision that the contract is stable, not a side effect of the next
breaking change.

## Alternatives Considered

| Alternative | Reason not chosen |
|---|---|
| Apply ADR-0007 word for word: the next breaking change makes `1.0.0` | Claims a stability the contract does not have yet, and uses up `1.0.0` on an arbitrary change instead of a decision. |
| Release a breaking change as PATCH | PATCH promises no break. A user who pins `~0.1` would get a behaviour change with no warning. |
| Keep ADR-0007 silent and decide each time | Leaves the same gap open at every release. |

## Consequences

- **Positive:** A PATCH bump is always safe to take, before and after 1.0.
- **Positive:** `1.0.0` stays free to mean "the contract is stable".
- **Negative:** Before 1.0, a MINOR bump can be a new feature or a break; the
  user must read the changelog for `**BREAKING:**` entries on every `0.y.0`
  release.

## References

- [ADR-0007](0007-keep-a-changelog.md), the Semantic Versioning rule this applies before `1.0.0`.
- [`../agents/changelog.md`](../agents/changelog.md), step 3 of "Releasing".
- [Semantic Versioning 2.0.0, item 4](https://semver.org/spec/v2.0.0.html#spec-item-4).
