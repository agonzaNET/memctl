# Plan — SPEC-0011 Instance scope

Spec plan for [SPEC-0011](spec.md) (Accepted on 2026-11-23, R1–R13), governed
by ADR-0025. It stands alone, so it is a spec plan, not a stage plan
(ADR-0018). No requirement is deferred.

Exit: every requirement has a passing test that names it (`SPEC0011_Rn`),
`make verify` is green, and the maintainer's live check in Claude Desktop
(acceptance criteria R13 and Live) passes; then the spec is `Implemented`.
The release is decided separately.

## File map

| File | Change | Why |
|---|---|---|
| `cmd/memctl/scope.go` | create | `parseScope(flagValue string, flagSet bool, getenv)`: precedence, trimming, the 200-character, UTF-8 and control-character rules, errors naming the source (R1–R3; ADR-0002: own file for its own rule) |
| `cmd/memctl/scope_test.go` | create | Unit tests of `parseScope`; binary tests for exit status 2 and the `started` field (R1–R4) |
| `cmd/memctl/config.go` | modify | `--scope` flag, `scope` field of `mcpConfig`, call to `parseScope` (R1) |
| `cmd/memctl/main.go` | modify | Usage text names `--scope` (or `MEMCTL_SCOPE`) (SPEC-0004 R1) |
| `cmd/memctl/mcp.go` | modify | Pass the scope to the server and to the owner's handshake options; `scope` on `started` only when set (R4, R10, R11) |
| `internal/mcpserver/tools.go` | modify | `scopePrefix` constant between the agent-facing markers (R9) |
| `internal/mcpserver/scope.go` | create | `withScope(text, scope string) string`: the fixed text, a line break and the scope line, or the text unchanged without a scope (R5–R7) |
| `internal/mcpserver/server.go` | modify | `Options.Scope`; the instructions through `withScope` (R5, R7) |
| `internal/mcpserver/{read,search,context,write,edit,move,delete}.go` | modify | Each `Description:` through `withScope` (R6, R7) |
| `internal/mcpserver/scope_test.go` | create | In-memory SDK client sessions with and without a scope: instructions, `tools/list`, schemas, annotations, one read and one write result, the `tool call` line (R5–R8) |
| `internal/share/hello.go` | modify | `Scope string \`json:"scope,omitempty"\`` on `Hello` and `Answer`; `NewHello` takes the scope (R11) |
| `internal/share/listen.go` | modify | `Owner.Scope`; the `ok` answer carries it (R11) |
| `internal/share/hello_test.go`, `greet_test.go` | modify | Scope present only when set; hello and answer without `scope` accepted; unknown field accepted; protocol 1 (R11) |
| `cmd/memctl/forward.go` | modify | Hello with the forwarder's scope; `WARN` `scope differs` with `owner_scope` (R11, R12) |
| `cmd/memctl/owner.go` | modify | `WARN` `scope differs` with `forwarder_pid` and `forwarder_scope` (R12) |
| `cmd/memctl/share_e2e_test.go` | modify | Owner with a scope, forwarder without: the forwarded session sees the owner's line; one `WARN` on each side; equal scopes log nothing (R10, R12) |
| `mcpb/manifest.json` | modify | Optional `user_config.scope` (string, default `""`) and `mcp_config.env.MEMCTL_SCOPE` (R13) |
| `scripts/mcpb-build.test.sh` | modify | Cases: `scope` optional string with default `""`; `MEMCTL_SCOPE` mapped to `${user_config.scope}`; args unchanged (R13) |
| `README.md` | modify | Flag table row; a "Several notes roots" section with a work and a personal example; the extension's Scope field |
| `CHANGELOG.md` | modify | `[Unreleased]` → `Added`: `--scope` / `MEMCTL_SCOPE`, the extension's Scope field |
| `docs/specs/0004-mcp-read-tools/spec.md` | modify | Amendment note: the Inputs table gains `--scope`; R5 and R13 extended by SPEC-0011 |
| `docs/specs/0007-write-tools/spec.md` | modify | Amendment note: the write tool descriptions carry the scope line (SPEC-0011 R6) |
| `docs/specs/0009-shared-instance/spec.md` | modify | Amendment note: hello and answer gain `scope`; `WARN` `scope differs` (SPEC-0011 R11, R12) |
| `docs/specs/0011-instance-scope/spec.md`, `docs/specs/README.md` | modify | Acceptance boxes; `Implemented` after the live check |

No new third-party module; `parseScope` uses `strings`, `unicode` and
`unicode/utf8`. The go-sdk stays in `internal/mcpserver` (ADR-0010):
`Options.Scope` is a plain string.

## Dependency order

```
T1 → T3            (flag before wiring)
T2 → T3            (server option before wiring)
T4 → T5            (handshake fields before the warning)
T3 → T5            (owner and forwarder need the parsed scope)
T6                 (independent)
T3, T5, T6 → T7 → T8 → T9
```

## Tasks

Each task: test first, named `SPEC0011_Rn`; see it fail; commit the failing
test; implement; full suite green; commit (`feat(scope): … (spec: SPEC-0011)`).

- [x] **T1 — Parse the scope (R1, R2, R3).** `parseScope` and the `--scope`
  flag. Table: flag only, variable only, both, empty flag over a set
  variable, surrounding white space, only white space; 200 code points with
  multi-byte characters accepted, 201 refused; `\n`, `\t`, invalid UTF-8
  refused; the error names `--scope` or `MEMCTL_SCOPE` and the rule, not the
  value. Usage text names the flag.
- [x] **T2 — Scope line in the agent-facing text (R5, R6, R7, R8, R9).**
  `scopePrefix` between the markers; `withScope`; `Options.Scope` applied to
  the instructions and the eight descriptions. Tests with an in-memory client:
  with a scope, instructions and every description are fixed text + `\n` +
  line; without, they equal the constants; schemas, annotations, a `note_read`
  and a `note_write` result and the `tool call` line equal with and without.
  `scripts/check-docs.sh` passes.
- [x] **T3 — Wire it through `memctl mcp` (R3, R4, R5, R6).** Binary tests: a
  bad `--scope` and a bad `MEMCTL_SCOPE` exit 2 with the rule on stderr;
  `started` has `scope` only when set; an SDK client of the binary started
  with `--scope` sees the line in the instructions and in `tools/list`.
- [x] **T4 — Scope in the handshake (R11).** `Hello.Scope`, `Answer.Scope`
  with `omitempty`; `NewHello(version, pid, author, scope)`; `Owner.Scope` in
  the `ok` answer. Tests: JSON has `scope` only when set; a hello and an answer
  without it, and with an unknown field, are accepted; `protocol` stays 1.
- [x] **T5 — Shared sessions and the warning (R10, R12).** Owner and
  forwarder compare scopes after an `ok` and log `WARN` `scope differs`.
  Multi-process tests: owner with a scope, forwarder without; the forwarded
  session sees the owner's line; one `WARN` on each side with
  `forwarder_pid` / `forwarder_scope` and `owner_scope`; the session works;
  equal scopes log no `scope differs`.
- [x] **T6 — Extension field (R13).** Manifest `user_config.scope` and
  `mcp_config.env.MEMCTL_SCOPE`; `scripts/mcpb-build.test.sh` cases; `make
  mcpb` builds.
- [x] **T7 — Docs.** README, CHANGELOG `[Unreleased]`, amendment notes in
  SPEC-0004, SPEC-0007 and SPEC-0009. `scripts/check-docs.sh` passes.
- [x] **T8 — Verify.** `make verify` (tests, vet, gofmt, check-docs, script
  tests); a sample run: `memctl mcp --scope …` over a scratch root answers
  `initialize` and `tools/list` with the line; acceptance boxes R1–R12 ticked
  against the tests that prove them.
- [x] **T9 — Live check (maintainer) and close (R13, Live).** Alberto sets a
  scope on `infradocs` and `albdocs` in Claude Desktop (and installs a bundle
  built with `make mcpb` with and without a scope); a session linked through
  the remote-devices bridge sees each instance's line when it loads its tools.
  Then tick R13 and Live, set SPEC-0011 `Implemented`, update the index.

## Definition of done

- `make verify` passes; output pasted as evidence.
- Every requirement R1–R13 has a passing test named `SPEC0011_Rn` (R13 by the
  `mcpb-build` script test plus the live check).
- Every acceptance box in the spec is ticked against a test or the live check.
- `CHANGELOG.md` has the `[Unreleased]` entries; SPEC-0004, SPEC-0007 and
  SPEC-0009 carry their amendment notes; the index matches every status.
