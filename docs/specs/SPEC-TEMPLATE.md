# SPEC-NNNN - <Title — noun phrase naming the capability>

## Status

Draft

<!--
Status values: Draft | Accepted | Implemented | Superseded by SPEC-NNNN | Deprecated
Lifecycle: Draft (under review) → Accepted (approved; the code may be built,
but at least one acceptance criterion, for example a live test, is still
open) → Implemented (shipped; code satisfies this spec and all acceptance
criteria are ticked). Superseded by SPEC-NNNN and Deprecated are terminal.
Governing ADRs: list the ADR(s) that justify the decisions in this spec.
-->

**Governing ADRs:** ADR-NNNN, …

## Purpose

One or two sentences: what this capability is for and who/what relies on it.

## Scope

**In scope**

- What this spec covers.

**Out of scope**

- What it deliberately does not cover (and which spec covers it instead, if any).

## Inputs

What the capability consumes: tool arguments, command-line flags, environment
variables, files under the notes root, external command output, etc. Be explicit about
required vs optional and about defaults.

## Outputs

What the capability produces: tool results and error codes, files written,
git commits, log lines, process exit behaviour. Give concrete examples.

## Behaviour & Requirements

Numbered, testable requirements. Use MUST / MUST NOT / SHOULD deliberately.

- **R1** — …
- **R2** — …

## Error handling

How failure modes are detected and surfaced. Map each failure to its observable
result (error code, exit status, log line).

## Acceptance criteria — "Done looks like"

Concrete, checkable statements, each citing the requirement(s) it proves. A
reviewer can verify each against a running `memctl` or the code, and every
requirement above has at least one box here.

- [ ] **R1** — …
- [ ] **R2** — …

## References

- Related specs and ADRs.
