# ADR-0001 - Record architecture decisions

## Status

Accepted

**Area:** Methodology

## Context

As the project grows, decisions about technology choices, trade-offs, and constraints
get lost. New contributors — human or AI — lack the context to understand why things are
the way they are, leading to repeated debates or accidental reversals of past decisions.

## Decision

We will record significant architectural decisions as ADRs in `docs/adr/`.
Each ADR follows the template in `TEMPLATE.md` and is committed alongside the
code change it describes.

## Consequences

- **Positive:** Decisions are visible, searchable, and version-controlled.
- **Positive:** Writing an ADR creates a lightweight forcing function to articulate reasoning.
- **Negative:** The ADR index in `docs/adr/README.md` must be kept up to date when
  new ADRs are added, or an ADR's status or area changes. `scripts/check-docs.sh`
  enforces this (ADR-0009).
