# ADR-0002 - Code modularity

## Status

Accepted

**Area:** Methodology

## Context

Most languages let an entire program live in a single file. As the codebase grows, a
decision is needed on when and how to split code across files and modules. Two opposing
pressures exist: keeping the file count low to reduce navigation overhead, and splitting
code so that each unit is narrow enough to read without scrolling past unrelated logic.

## Decision

Split by responsibility, not by line count. Each file, module, or package should have a
single, nameable concern. Prefer more files over longer files when combining them would
force a reader to hold unrelated context in their head.

New behaviour follows the same pattern: add a file (or module/package) with a name that
describes exactly what it does. Do not append unrelated logic to an existing file because
it happens to be small. The concrete module boundaries for this project's stack are
recorded in the relevant guide under `docs/agents/` and in the Project Layout section of
`CLAUDE.md`.

## Consequences

- **Positive:** Each file can be read and understood on its own without scrolling past unrelated code.
- **Positive:** The file name is self-documenting — it sets the expectation before the first line is read.
- **Positive:** Merge conflicts are less likely because unrelated changes land in separate files.
- **Negative:** A larger file count means more navigation; this is accepted as the lesser cost compared to files that mix concerns.
- **Negative:** Small modules (one file, a handful of types) may feel over-engineered at first; the structure is chosen for the trajectory of the codebase, not its current size.
