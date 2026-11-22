# ADR-0023 - Serve MCP over streamable HTTP from a long-running `memctl serve`, on loopback only and without authentication (amends ADR-0014, ADR-0021)

## Status

Proposed

> **Deferred (2026-11-22).** This ADR stays `Proposed`, and its implementation
> is deferred until further notice. After it was written, agentgateway was
> found to run stdio MCP servers itself as child processes (an `mcp` backend
> with a `stdio` target). The Raspberry Pi deployment can then reach
> `memctl mcp` through agentgateway with no HTTP transport in memctl. That
> path is tested first; this decision is taken up again only if it fails or
> another client needs HTTP.

**Area:** Lifecycle

## Context

Design D1 makes `memctl` a stdio server that the MCP client starts, with HTTP
listed under "Backlog after v2": streamable HTTP served by the owner of a root,
authenticated by a key that memctl generates (`.memory/token`), required even
on loopback, with a git author per key.

The maintainer now wants a notes root on a Raspberry Pi (`memory.goncas.net`),
used from clients that cannot start a process on that machine. The Pi already
runs Traefik behind a Cloudflare tunnel, Pocket ID as identity provider, and
plans agentgateway in front of MCP servers to do MCP OAuth with Pocket ID. That
deployment is waiting for an HTTP transport; semantic search (v2) is not, so
HTTP moves ahead of v2 as 0.4.0.

Forces:

- A service on the Pi has no stdio client. The owner's lifetime rule
  (ADR-0014 as amended by ADR-0021) ends the process when its own stdin
  session and every forwarded session have ended, which in a container is at
  once.
- Authentication for remote MCP clients is OAuth, and OAuth client
  registration depends on the authorization server (Pocket ID), not on
  memctl. agentgateway, which does that part, is a separate Rust program and
  cannot be built into memctl.
- The go-sdk in use (v1.8.0) has a streamable HTTP handler with a stateless
  mode, JSON responses, a request body limit and DNS-rebinding protection. The
  rebinding protection answers 403 to a request on loopback whose `Host` is
  not localhost, which is exactly what a proxy that keeps the public host name
  sends.
- memctl's tools are plain request and response: the server never calls the
  client and sends no notifications.
- Go's standard library has `http.CrossOriginProtection`, so cross-origin
  checks need no new module (ADR-0011).

## Decision

> We will add `memctl serve`, a long-running owner that serves the MCP tools
> over stateless streamable HTTP on a loopback address only, with no
> authentication of its own, because the proxy in front of it owns
> authentication, a loopback-only listener keeps the unauthenticated endpoint
> out of reach of anything but that proxy, and no tool needs sessions.

- **Command.** `memctl serve --root <notes> --http <loopback host:port>`, with
  `--git-author` and `--log-level` as for `memctl mcp`. `memctl mcp` is
  unchanged and takes no HTTP flag.
- **No authentication in memctl.** Whatever can reach the port can read and
  write the notes. Authentication is the job of the proxy in front (on the Pi,
  agentgateway in the same Podman pod). The generated key, `memctl token` and
  the per-key git author of the backlog item are dropped. Every write is
  committed with the one `--git-author`.
- **Loopback only.** The host of `--http` must be `127.0.0.1`, `::1` or
  `localhost` (which binds `127.0.0.1`); anything else is refused at start.
  Opening it up later is an additive change behind an explicit flag.
- **Host allow-list.** memctl replaces the SDK's rebinding check with its own:
  a request's `Host` must be a loopback name or one of the names given with
  the repeatable `--http-allowed-host`, or it gets 403 and a `WARN`
  `host refused` that names the host. A proxy that keeps the public name works
  without a `Host` rewrite, and a rebinding attacker's domain is still refused.
- **Cross-origin requests** that a browser marks as such get 403, through
  `http.CrossOriginProtection`.
- **Stateless, JSON responses.** `/mcp` answers each POST on its own, with no
  `Mcp-Session-Id`, no event stream and no resumption; GET and DELETE answer
  405. A restart of memctl never leaves a client with an unknown session.
  `GET /healthz` answers `200 ok` for container and proxy health checks.
- **Lifetime (amends ADR-0014 and ADR-0021).** `serve` takes the instance
  lock, loads the notes, listens on the Unix socket of ADR-0021 and on the
  HTTP address, never reads stdin, and runs until SIGINT or SIGTERM. Sessions
  coming and going never stop it. On a signal it stops accepting, lets the
  tool calls in progress finish, closes forwarded connections, makes the final
  commit and exits. When the lock is held, `serve` exits at once with the lock
  message; it never forwards.
- **Sharing.** A later `memctl mcp` on the same root forwards to `serve` as to
  any owner (ADR-0021). The `tool call` line of HTTP calls names the session
  `http`.
- **Packages.** The HTTP handler is built in `internal/mcpserver`, the only
  importer of go-sdk (ADR-0010); `cmd/memctl` owns the `http.Server`, the
  listener, the Host check and the lifecycle. No module is added.

## Alternatives Considered

| Alternative | Reason not chosen |
|---|---|
| Generated key in `.memory/token`, required even on loopback (the design backlog item) | The proxy does OAuth anyway and would only inject the key; the maintainer chose to leave authentication to the proxy and keep memctl free of token storage and rotation. |
| Both: memctl's key now, Pocket ID JWT checks later | More code paths for a single-user deployment whose port only the proxy can reach. |
| memctl validates OAuth tokens itself (protected-resource metadata, JWT checks against Pocket ID's keys) | Security-sensitive code memctl must keep current with the MCP authorization spec, and it does not remove the proxy unless Pocket ID supports the client registration MCP clients need, which is unverified. |
| Any listen address, with or without an opt-in flag | Without authentication, a listener on a reachable interface gives any network peer full write access; one misconfigured route is enough. |
| Keep the SDK's rebinding check and require the proxy to rewrite `Host` | A gateway that keeps the public host name fails with a bare 403 and no memctl log line. |
| Turn the rebinding check off | On a machine with a browser, a page on an attacker's domain that resolves to 127.0.0.1 could read and write the notes. |
| Stateful sessions with an event stream and an idle timeout | No tool needs server-to-client messages; sessions add expiry, memory, broken sessions after a restart, and stream buffering in proxies. |
| `--http` on `memctl mcp`, one command for stdio and HTTP | In a container stdin closes at once, so the command would need lifetime rules that depend on a flag; `mcp` and its tests would change for no stdio user. |
| A detached daemon started by the first client | Rejected in ADR-0021 for the same reasons; a service manager (systemd, Quadlet) already starts and stops `serve`. |

## Consequences

- **Positive:** The Pi deployment is unblocked with no new module and no change
  to `memctl mcp`.
- **Positive:** One owner per root still holds: HTTP clients, local stdio
  clients and forwarders share one index and one git committer.
- **Positive:** `serve` fits a service manager: started once, stopped by a
  signal, health-checked on `/healthz`.
- **Negative:** memctl alone is not safe to expose. Its safety rests on the
  loopback-only listener and on the proxy; the README and the `serve` help
  must say so.
- **Negative:** Every change over HTTP is committed under one author; who made
  it is known only to the proxy.
- **Negative:** A remote setup needs the proxy in the same network namespace
  (a Podman pod, or the same host).
- **Note:** Stateless mode follows the MCP spec's move away from sessions
  (SEP-2567); a later tool that needs server-to-client messages would need a
  new decision.

## References

- Design D1, D12, D13, D21, "Backlog after v2" in
  [`../design/solution-design.md`](../design/solution-design.md).
- [ADR-0014](0014-shutdown-flushes-pending-commits.md) and
  [ADR-0021](0021-shared-instance-over-unix-socket.md), amended here.
- [ADR-0010](0010-one-importer-per-third-party-module.md),
  [ADR-0011](0011-mcp-sdk-and-yaml-v3.md) (no new module).
- SPEC-0010 (`memctl serve`), SPEC-0009 (shared instance), SPEC-0006 (lock).
- go-sdk v1.8.0 `mcp.StreamableHTTPOptions` (`Stateless`, `JSONResponse`,
  `DisableLocalhostProtection`, `MaxRequestBodyBytes`).
- MCP security best practices, local server compromise and DNS rebinding:
  <https://modelcontextprotocol.io/specification/2025-11-25/basic/security_best_practices#local-mcp-server-compromise>.
- MCP client registration (CIMD preferred, DCR deprecated):
  <https://modelcontextprotocol.io/specification/latest/basic/authorization/client-registration>.
