# SPEC-0010 - `memctl serve`: MCP over stateless streamable HTTP on loopback

## Status

Draft

**Governing ADRs:** ADR-0002, ADR-0005, ADR-0010, ADR-0011, ADR-0014, ADR-0016, ADR-0021, ADR-0023

## Purpose

Let MCP clients that cannot start a process on the notes machine use a notes
root: `memctl serve` runs as a long-lived owner that serves the tools over
stateless streamable HTTP on a loopback address. Authentication is left to a
proxy in front of it, such as agentgateway in the same Podman pod on the
maintainer's Raspberry Pi (ADR-0023).

## Scope

**In scope**

- The subcommand `memctl serve`, its flags and environment variables.
- The loopback-only listener, the `Host` allow-list and the cross-origin check.
- `POST /mcp` as stateless streamable HTTP with JSON responses, and
  `GET /healthz`.
- Start, the instance lock, the Unix socket of SPEC-0009, and shutdown on a
  signal.
- The log lines `serve` adds or changes.

**Out of scope**

- Authentication and TLS: the job of the proxy in front (ADR-0023).
- Listening on a non-loopback address.
- Stateful sessions, event streams, stream resumption and server-to-client
  requests.
- `memctl mcp`, which is unchanged (SPEC-0004, SPEC-0009) apart from R5.
- A git author per HTTP client: every write uses `--git-author` (SPEC-0008).
- Container images, Quadlet units and proxy configuration.

## Inputs

**Command line.** `memctl serve --root DIR --http ADDR [--http-allowed-host NAME]... [--git-author AUTHOR] [--log-level LEVEL]`.

| Flag | Environment variable | Default | Rule |
|---|---|---|---|
| `--root` | `MEMCTL_ROOT` | none (required) | As SPEC-0004 and SPEC-0003 R1 |
| `--log-level` | `MEMCTL_LOG_LEVEL` | `info` | As SPEC-0004 |
| `--git-author` | `MEMCTL_GIT_AUTHOR` | `memctl <memctl@localhost>` | As SPEC-0008 |
| `--http` | `MEMCTL_HTTP` | none (required) | `host:port`; host `127.0.0.1`, `[::1]` or `localhost`; port 0–65535, where 0 picks a free port |
| `--http-allowed-host` | `MEMCTL_HTTP_ALLOWED_HOSTS` | none | Repeatable flag; the variable is a comma-separated list. Each value a host name or IP address with no port, scheme, path or space |

A flag wins over its environment variable; given `--http-allowed-host` flags
replace the variable's list.

**HTTP requests.** `POST /mcp` with a JSON-RPC message (MCP streamable HTTP),
`GET /healthz`. Requests carry a `Host` header and, from browsers, `Origin` and
`Sec-Fetch-Site`.

**Signals.** SIGINT and SIGTERM. Stdin is not read.

## Outputs

**HTTP responses.**

| Request | Response |
|---|---|
| `POST /mcp`, request | `200`, `Content-Type: application/json`, one JSON-RPC response |
| `POST /mcp`, notification or response | `202`, no body |
| `GET` or `DELETE /mcp` | `405` |
| `GET /healthz` | `200`, `Content-Type: text/plain; charset=utf-8`, body `ok\n` |
| Any other path or method | `404` |
| `Host` not allowed (R16) | `403`, body `host not allowed\n` |
| Cross-origin browser request (R17) | `403` |
| Body over 4 MiB (R18) | `413` |

No response carries an `Mcp-Session-Id` header.

**Log lines** (JSON on stderr):

- `started` gains `http`, the bound address with the real port, for example
  `"http":"127.0.0.1:8080"`; `sharing` as SPEC-0009 R3.
- `tool call` lines of HTTP calls carry `"session":"http"`.
- `WARN` `host refused` with `host`.
- `ERROR` `cannot listen` with `address` and `reason`.
- `stopped`, as SPEC-0004 R6.

**Exit status.** 0 after a signal (R11); 1 when the lock is held (R6), the
address cannot be bound (R7) or the root is rejected (SPEC-0004 R2); 2 for a
flag error (R1–R4).

## Behaviour & Requirements

### Command and configuration

- **R1** — `memctl serve` MUST take `--root`, `--log-level` and `--git-author`
  and their variables with the rules and messages of `memctl mcp`
  (SPEC-0004 R1, R2; SPEC-0008), and `--http` and `--http-allowed-host` and
  their variables. A positional argument, an unknown flag or a bad value MUST
  print usage to stderr and exit with status 2, naming the flag and the rule
  (SPEC-0004 R1).
- **R2** — Without `--http` and `MEMCTL_HTTP`, `serve` MUST exit with status 2
  and `--http (or MEMCTL_HTTP) is required`.
- **R3** — The host of `--http` MUST be `127.0.0.1`, `::1` (written
  `[::1]:port`) or `localhost`, compared without case. Any other host,
  including an empty host, `0.0.0.0`, `::`, another address or another name,
  MUST exit with status 2 and
  `--http must name a loopback address (127.0.0.1, [::1] or localhost)`.
  `localhost` MUST bind `127.0.0.1` without a name lookup.
- **R4** — Each `--http-allowed-host` value, and each element of
  `MEMCTL_HTTP_ALLOWED_HOSTS` after trimming spaces (empty elements are
  ignored), MUST be a host name or IP address with no port, scheme, path or
  space; otherwise `serve` MUST exit with status 2 and
  `--http-allowed-host must be a host name without port`.
- **R5** — `memctl mcp` MUST refuse `--http` and `--http-allowed-host` as
  unknown flags (SPEC-0004 R1).

### Start and lifetime

- **R6** — `serve` MUST take the instance lock (SPEC-0006 R1–R4). When the lock
  is held, it MUST log the message of SPEC-0006 R5 and exit with status 1,
  without loading the notes, without forwarding and without retrying.
- **R7** — After the startup load (SPEC-0003 R8), `serve` MUST listen on the
  Unix socket as an owner (SPEC-0009 R1–R3) and bind the `--http` address. It
  MUST NOT accept an HTTP connection or a socket connection before the load is
  complete. When the address cannot be bound, it MUST log one `ERROR`
  `cannot listen` with the address and the reason, close the socket listener,
  and exit with status 1 without a `started` line.
- **R8** — The `started` line MUST carry `http` with the bound address, with the
  real port when the given port was 0, and `sharing` (SPEC-0009 R3), in
  addition to the fields of SPEC-0004 R3 and SPEC-0008.
- **R9** — `serve` MUST NOT read stdin and MUST NOT write to stdout. A closed
  stdin, stdout or stderr MUST NOT stop it (SPEC-0009 R16).
- **R10** — `serve` MUST keep running while no session is open. The end of a
  forwarded session or of an HTTP request MUST NOT start the shutdown.
- **R11** — On SIGINT or SIGTERM, `serve` MUST start the shutdown at once: stop
  accepting HTTP and socket connections, let the tool calls in progress finish
  and send their responses, start no new tool call, close every forwarded
  connection, make the final commit (SPEC-0008 R12), log `stopped` and exit
  with status 0. A second signal MUST behave as SPEC-0008 R13.
- **R12** — A later `memctl mcp` on the same root MUST forward to `serve` as to
  any owner (SPEC-0009 R4–R11), and its tool calls MUST be logged with
  `forward-<n>` (SPEC-0009 R13).

### HTTP

- **R13** — `POST /mcp` MUST be served as stateless streamable HTTP with JSON
  responses: no `Mcp-Session-Id` header is set or read, every request is
  served by a fresh session of the one MCP server, and results are
  `application/json`, never `text/event-stream`. `GET` and `DELETE` on `/mcp`
  MUST answer `405`.
- **R14** — HTTP requests MUST use the owner's note service and git committer
  (SPEC-0009 R12): their tool calls are serialised with every other session's
  by the note service mutex (SPEC-0004 R7), and their writes are committed
  with the owner's git author (SPEC-0008 R7).
- **R15** — `GET /healthz` MUST answer `200` with the body `ok\n` and MUST NOT
  touch the notes or take the note service mutex. Any other path, and any other
  method on `/healthz`, MUST answer `404`.
- **R16** — Every request MUST pass the host check before any other handling:
  the host part of the `Host` header, without its port and compared without
  case, MUST be `localhost`, `127.0.0.1`, `::1` or one of the allowed hosts
  (R4). Otherwise `serve` MUST answer `403` with the body `host not allowed\n`
  and log one `WARN` `host refused` with `host`, the header value cut to 255
  bytes. The SDK's own localhost check MUST be off, since this check replaces
  it.
- **R17** — After the host check, a request that a browser marks as
  cross-origin MUST be refused with `403` (`http.CrossOriginProtection`). A
  request with neither `Origin` nor `Sec-Fetch-Site` MUST pass.
- **R18** — A request body over 4 MiB MUST be refused with `413`.
- **R19** — Each tool call served over HTTP MUST log the `tool call` line of
  SPEC-0004 R12 with `"session":"http"`. `serve` MUST write no other line per
  request, except R16's `WARN`.

## Error handling

| Failure | Result |
|---|---|
| Missing or bad flag or variable | Usage and message on stderr, exit 2 (R1–R4) |
| Root rejected | One `ERROR`, exit 1 (SPEC-0004 R2) |
| Lock held by another process | Message of SPEC-0006 R5, exit 1, no forwarding (R6) |
| Socket cannot be created | `WARN` `sharing is off`, HTTP serves on (SPEC-0009 R2) |
| Address in use or not bindable | `ERROR` `cannot listen`, exit 1 (R7) |
| `Host` not allowed | `403`, `WARN` `host refused` (R16) |
| Cross-origin browser request | `403` (R17) |
| Body over 4 MiB | `413` (R18) |
| Malformed JSON-RPC, unknown tool, tool error | As the SDK and SPEC-0004/SPEC-0007 define, in the JSON response |
| SIGINT or SIGTERM | Final commit, `stopped`, exit 0 (R11) |

## Acceptance criteria — "Done looks like"

- [ ] **R1** — `memctl serve` with each of the shared flags and variables behaves as `memctl mcp`; a positional argument and an unknown flag exit 2 with usage.
- [ ] **R2** — `memctl serve --root <dir>` without `--http` exits 2 with the message.
- [ ] **R3** — `--http 0.0.0.0:8080`, `--http [::]:8080`, `--http 192.168.1.5:8080`, `--http example.org:8080` and `--http :8080` exit 2; `127.0.0.1:0`, `[::1]:0` and `localhost:0` start, and `localhost` is bound on `127.0.0.1`.
- [ ] **R4** — `--http-allowed-host memory.example.org:443`, `https://memory.example.org` and a value with a space exit 2; `MEMCTL_HTTP_ALLOWED_HOSTS=" a.example , ,b.example"` allows `a.example` and `b.example`; a flag replaces the variable's list.
- [ ] **R5** — `memctl mcp --http 127.0.0.1:0` exits 2 as an unknown flag.
- [ ] **R6** — With an owner already on the root, `memctl serve` exits 1 with the lock message, logs no `notes loaded` line and opens no connection to the socket.
- [ ] **R7** — With the port taken, `serve` logs `cannot listen` and exits 1 with no `started` line and no socket file left; no request is answered before `started`.
- [ ] **R8** — With port 0, `started` carries the real bound address and `sharing`.
- [ ] **R9** — `serve` started with stdin closed (`</dev/null`) keeps serving; nothing is written to stdout.
- [ ] **R10** — After a forwarded session and several HTTP requests have ended, `serve` still answers `/healthz`.
- [ ] **R11** — SIGTERM during a slow tool call: the call's response arrives, the queued write is committed, `stopped` is logged, exit 0; a second signal exits 1 as SPEC-0008 R13.
- [ ] **R12** — A `memctl mcp` on the same root forwards to `serve`; its tool call is logged with `forward-1`.
- [ ] **R13** — An `initialize`, `tools/list` and `tools/call` over `POST /mcp` each get a `200` JSON response without `Mcp-Session-Id`; a `tools/call` without a prior `initialize` succeeds; `GET /mcp` answers `405`.
- [ ] **R14** — A `note_write` over HTTP and one through a forwarder are both committed with the owner's git author, and neither call overlaps the other.
- [ ] **R15** — `GET /healthz` answers `200 ok` while a tool call holds the note service mutex; `GET /other` and `POST /healthz` answer `404`.
- [ ] **R16** — `Host: memory.example.org` answers `403` and logs `host refused` without `--http-allowed-host`, and `200` with `--http-allowed-host memory.example.org` (also as `MEMORY.example.org:443`); `Host: localhost:8080` and `Host: 127.0.0.1` pass.
- [ ] **R17** — A request with `Sec-Fetch-Site: cross-site` answers `403`; the same request without browser headers answers `200`.
- [ ] **R18** — A 5 MiB body answers `413`.
- [ ] **R19** — A `tools/call` over HTTP logs one `tool call` line with `"session":"http"` and no other line.
- [ ] **Live check** — On the maintainer's machine: Claude Code or the MCP Inspector, pointed at `serve` through a proxy that keeps the public host name, lists and calls the tools; a write appears as one git commit.

## References

- ADR-0023 (this transport), ADR-0021 (shared instance), ADR-0014 (shutdown).
- SPEC-0003 (startup load), SPEC-0004 (`memctl mcp`, tool-call log), SPEC-0006 (lock), SPEC-0008 (git committer), SPEC-0009 (shared instance).
- Design D1, D21 and "Deployment and config" in [`../../design/solution-design.md`](../../design/solution-design.md).
