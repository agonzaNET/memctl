# SPEC-0011 - Instance scope: one line that tells the agent what a notes root holds

## Status

Accepted

<!--
Status values: Draft | Accepted | Implemented | Superseded by SPEC-NNNN | Deprecated
Lifecycle: Draft (under review) → Accepted (approved; the code may be built,
but at least one acceptance criterion, for example a live test, is still
open) → Implemented (shipped; code satisfies this spec and all acceptance
criteria are ticked). Superseded by SPEC-NNNN and Deprecated are terminal.
Governing ADRs: list the ADR(s) that justify the decisions in this spec.
-->

**Governing ADRs:** ADR-0002, ADR-0005, ADR-0013, ADR-0016, ADR-0021, ADR-0024, ADR-0025

## Purpose

Let the person who configures a `memctl mcp` instance say in one line what its
notes root holds, and show that line to the agent in the server instructions
and in every tool description, so that an agent with several instances (for
example `infradocs` for work and `albdocs` for personal notes) knows which one
to search and which one to write to (ADR-0025).

## Scope

**In scope**

- `cmd/memctl`: the `--scope` flag and `MEMCTL_SCOPE`, their rules, and the
  `scope` field of the `started` log line.
- `internal/mcpserver`: the scope line appended to the server instructions and
  to the eight tool descriptions.
- `internal/share` and `cmd/memctl`: the `scope` field of the handshake and the
  `WARN` `scope differs` on the owner and the forwarder.
- `mcpb/manifest.json`: an optional `scope` field of the Claude Desktop
  extension, passed as `MEMCTL_SCOPE` (ADR-0024).
- `README.md`: how to set a scope.

**Out of scope**

- Enforcing the scope: `memctl` does not judge what a note is about; a note
  written to the wrong instance is moved by hand.
- `memctl check`, which has no MCP surface.
- `memctl serve` (SPEC-0010, deferred), which takes the same flag when it is
  built.
- Changing the fixed instructions or descriptions (SPEC-0004, SPEC-0007).

## Inputs

**Command line.** `memctl mcp` gains one optional flag, beside those of
SPEC-0004 and SPEC-0008:

| Flag | Environment variable | Default | Rule |
|---|---|---|---|
| `--scope` | `MEMCTL_SCOPE` | none (no scope) | One line: at most 200 characters (Unicode code points) after leading and trailing white space is removed, valid UTF-8, no control character (line break, tab, …). Empty or only white space means no scope. |

A flag wins over its environment variable, also when the flag is empty
(`--scope ""` turns off a scope set in `MEMCTL_SCOPE`).

**Handshake.** The forwarder's hello and the owner's `ok` answer of SPEC-0009
R6 each gain an optional `scope` string.

**Extension.** `mcpb/manifest.json` gains an optional `user_config` field
`scope` (type `string`, not required, default `""`), passed to the server as
the environment variable `MEMCTL_SCOPE`.

## Outputs

**Server instructions and tool descriptions.** With the scope
`Alberto's personal notes: family, homelab, hobbies. No appliedAI work.`, the
instructions and each of the eight tool descriptions end with the fixed text
of today, a line break, and:

```text
Scope of this memory, set by the person who runs it: Alberto's personal notes: family, homelab, hobbies. No appliedAI work.
```

Without a scope, they are exactly the fixed text of today.

**Handshake.**

```text
→ {"memctl":"forward","protocol":1,"version":"0.4.0","pid":4711,"git_author":"memctl <memctl@localhost>","scope":"Alberto's personal notes. No appliedAI work."}
← {"ok":true,"pid":4242,"version":"0.4.0","scope":"Alberto's personal notes. No appliedAI work."}
```

**Log lines** (JSON on stderr):

- `started` gains `scope` (the scope text) when a scope is set; no `scope`
  field otherwise.
- Owner: `WARN` `scope differs` with `forwarder_pid` and `forwarder_scope`.
- Forwarder: `WARN` `scope differs` with `owner_scope`.
- `forwarder_scope` and `owner_scope` are `""` for a side without a scope.

**Exit status.** A bad scope is a usage error, status 2 (SPEC-0004 R1).

## Behaviour & Requirements

### Flag and variable

- **R1** — `memctl mcp` MUST read the scope from `--scope`, or from
  `MEMCTL_SCOPE` when the flag is not given. A given flag MUST win over the
  variable, also when the flag's value is empty.
- **R2** — The scope MUST be the value with leading and trailing white space
  removed. An empty result MUST mean no scope.
- **R3** — A scope longer than 200 characters (Unicode code points), one that
  is not valid UTF-8, or one that contains a control character (Unicode
  category Cc, which includes line breaks and tabs) MUST be refused as a usage
  error with status 2 (SPEC-0004 R1). The error MUST name the flag or the
  variable that gave the value and the rule, and MUST NOT echo the value.
- **R4** — The `started` line MUST carry `scope` with the scope text when a
  scope is set, and MUST NOT carry a `scope` field otherwise. No other log line
  of SPEC-0004 or SPEC-0007 MAY change.

### Agent-facing text

- **R5** — With a scope, the server instructions MUST be the fixed
  instructions, a line break, then the scope line
  `Scope of this memory, set by the person who runs it: <scope>`.
- **R6** — With a scope, each of the eight tool descriptions (`note_read`,
  `search`, `context`, `list`, `note_write`, `note_edit`, `note_move`,
  `note_delete`) MUST be its fixed description, a line break, then the same
  scope line as R5.
- **R7** — Without a scope, the server instructions and every tool description
  MUST be byte for byte the fixed text.
- **R8** — The scope MUST NOT change any tool's name, input schema,
  annotations, results, error codes or `tool call` log line.
- **R9** — The fixed part of the scope line,
  `Scope of this memory, set by the person who runs it: `, MUST sit between
  the agent-facing markers in `internal/mcpserver/tools.go` and pass check 5 of
  `scripts/check-docs.sh` (ADR-0013).

### Shared instance (SPEC-0009)

- **R10** — Every session the owner serves, its own and every forwarded one,
  MUST get the owner's instructions and descriptions, so the owner's scope
  applies to all of them.
- **R11** — The forwarder's hello MUST carry `scope` when the forwarder has a
  scope and MUST omit it otherwise. The owner's `ok` answer MUST carry `scope`
  when the owner has a scope and MUST omit it otherwise. The protocol MUST stay
  `1`: a hello or answer without `scope` means no scope, and an unknown field
  MUST NOT make a handshake fail.
- **R12** — When the owner's and the forwarder's scopes differ (one side
  without a scope counts as `""`), the owner MUST log one `WARN`
  `scope differs` with `forwarder_pid` and `forwarder_scope`, the forwarder
  MUST log one `WARN` `scope differs` with `owner_scope`, and the session MUST
  go on. Equal scopes MUST log nothing.

### Extension

- **R13** — `mcpb/manifest.json` MUST declare an optional `user_config` field
  `scope` of type `string` with default `""` and pass it to the server as `MEMCTL_SCOPE` in
  `mcp_config.env`; an extension installed without a scope MUST start with no
  scope (R2).

## Error handling

| Failure | Result |
|---|---|
| `--scope` longer than 200 characters, not UTF-8, or with a control character | Usage to stderr naming `--scope` and the rule, status 2 (R3) |
| `MEMCTL_SCOPE` breaking the same rule, no `--scope` given | Same, naming `MEMCTL_SCOPE` (R3) |
| Forwarder and owner with different scopes | `WARN` `scope differs` on both sides; the session goes on with the owner's scope (R10, R12) |
| Old forwarder (no `scope` in its hello) and owner with a scope | Treated as a forwarder without a scope: `WARN` on the owner; the session goes on (R11, R12) |

## Acceptance criteria — "Done looks like"

- [x] **R1** — Table-driven test: flag only, variable only, both (flag wins),
  empty flag with a variable set (no scope).
- [x] **R2** — `--scope "  personal notes  "` gives `personal notes`;
  `--scope "   "` gives no scope.
- [x] **R3** — 200 characters (with multi-byte characters) accepted; 201
  refused; a line break, a tab and invalid UTF-8 refused; each error names the
  flag or variable and the rule, not the value; exit status 2 from the binary.
- [x] **R4** — `started` of a run with a scope has `scope` equal to the text;
  of a run without one has no `scope` key.
- [x] **R5** — An SDK client session against a server with a scope sees the
  instructions as fixed text + line break + scope line.
- [x] **R6** — The same session's `tools/list` shows all eight descriptions
  ending with the scope line, each with its fixed text unchanged before it.
- [x] **R7** — Without a scope, `tools/list` and the instructions equal the
  fixed constants.
- [x] **R8** — Schemas, annotations and the results and log line of one read
  and one write call are equal with and without a scope.
- [x] **R9** — `scripts/check-docs.sh` passes with the fixed part of the line
  between the markers.
- [x] **R10** — A forwarded session (forwarder started with no scope) against
  an owner with a scope sees the owner's scope line in instructions and
  descriptions.
- [x] **R11** — Hello and answer JSON carry `scope` only when set; a hello
  without `scope` and an answer without `scope` are accepted; protocol is `1`.
- [x] **R12** — Different scopes: one `WARN` `scope differs` on each side with
  the fields above, the session works; equal scopes: no such line.
- [ ] **R13** — `make mcpb` builds a bundle whose manifest has the optional
  `scope` field mapped to `MEMCTL_SCOPE`; live: the extension installed in
  Claude Desktop with and without a scope starts, and with a scope the line
  shows in the tool descriptions.
- [ ] **Live** — In Claude Desktop with `infradocs` and `albdocs` each given a
  scope, an agent session that is linked to the desktop through the
  remote-devices bridge sees each instance's scope line when it loads that
  instance's tools.

## References

- ADR-0025 (scope line), ADR-0013 (agent-first tool text), ADR-0021 (shared
  instance), ADR-0024 (Claude Desktop extension).
- SPEC-0004 (flags, R1 usage errors, R5 instructions, R13 tool text), SPEC-0007
  (write tool descriptions), SPEC-0009 (handshake, R6 to R8).
- Design D4 (one notes root per instance).
