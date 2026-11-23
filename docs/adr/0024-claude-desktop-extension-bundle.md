# ADR-0024 - Package memctl as a Claude Desktop extension (MCPB) with `make mcpb`

## Status

Accepted

> **Amended by [ADR-0025](0025-instance-scope-line.md)**: the manifest gains
> an optional `scope` field, passed to the server as `MEMCTL_SCOPE`.

> Revised on 2026-11-23, before the bundle was released, after the live
> check and the v0.4.0 release audit (W3): `display_name` is `NAME`, not
> `memctl (NAME)`, because Desktop names the server after it; the bundle
> carries an icon.

**Area:** Build

## Context

Claude Desktop starts `memctl mcp` from a hand-written entry in
`claude_desktop_config.json`: an absolute path to the binary and an absolute
`--root`. The maintainer runs two such entries side by side (two notes roots,
each under its own server name). Editing the JSON by hand is the only install
path, and the binary has to be installed separately.

Claude Desktop also installs MCP Bundles (`.mcpb`, the MCPB format of the
Model Context Protocol project, formerly DXT): a zip with a `manifest.json`
and the server files. A manifest can declare a `binary` server and a
`user_config` field of type `directory`, which Desktop asks for with a folder
picker and substitutes into the server arguments as `${user_config.<key>}`.
The bundle is installed by opening the file; the root can be changed later in
the extension's settings.

The bundle is a packaging artifact with no Go code, so per the workflow guide
it is governed by an ADR, not a spec. The maintainer approved the approach on
2026-11-23 (a manifest in the repository and a `make mcpb` target).

## Decision

> We will keep an MCPB manifest in `mcpb/manifest.json` and build a bundle
> holding the darwin/arm64 binary with `make mcpb`, so that memctl installs in
> Claude Desktop by opening one file and the notes root is chosen with a
> folder picker.

- `mcpb/manifest.json`: `manifest_version` `0.3`, server type `binary`,
  command `${__dirname}/server/memctl`, arguments
  `mcp --root ${user_config.root}`. One `user_config` field, `root`, of type
  `directory`, required, with no default. `compatibility.platforms` is
  `["darwin"]`, the only desktop platform with a release binary (ADR-0012).
- `scripts/mcpb-build.sh X.Y.Z NAME OUTDIR` builds the darwin/arm64 binary
  with `make build` (same flags and version injection as the release
  archives), stages it as `server/memctl` with `LICENSE` and `README.md`,
  sets `name` and `display_name` to `NAME` and `version` in the manifest
  with `jq`, adds `mcpb/icon.png` as the bundle's icon, and packs `OUTDIR/<NAME>_<X.Y.Z>_darwin_arm64.mcpb` with the MCPB CLI.
- `NAME` is the extension name and its display name. Claude Desktop names
  the server, and so the tool prefix, after the display name (default
  `memctl`). Two notes roots need two bundles with different names,
  for example `make mcpb MCPB_NAME=infradocs` and `MCPB_NAME=albdocs`:
  Desktop installs one copy of an extension per name.
- The MCPB CLI is run with `npx` and pinned (`@anthropic-ai/mcpb@2.1.2`);
  the `MCPB` make/environment variable overrides the command. Node.js is
  needed for this target only.
- `scripts/mcpb-build.test.sh` checks the staged bundle (manifest fields,
  binary present and executable) with a stub in place of the CLI, so
  `make verify` needs neither Node.js nor the network.
- The release workflow is unchanged: the bundle is built by hand, not
  published as a release asset.

## Alternatives Considered

| Alternative | Reason not chosen |
|---|---|
| Keep the hand-written `claude_desktop_config.json` entry only | Works, but needs the binary installed separately and absolute paths typed by hand; no folder picker. |
| Bundle pointing at an installed `memctl` (absolute `command`) | Not portable between users; the bundle would break when the binary moves. |
| A `default` root such as `${HOME}/notes` | A wrong default silently serves a folder the user did not pick; a required field with no default makes the choice explicit. |
| Publish the bundle from the release workflow | Adds Node.js and a new asset to the release path before the bundle has been tried; can be added later by its own ADR. |
| `manifest_version` `0.4` | Adds only `uv` servers; `0.3` is enough and works with more Claude Desktop versions. |

## Consequences

- **Positive:** memctl installs in Claude Desktop by opening one file; the
  root is picked and later changed in the extension's settings.
- **Positive:** the bundle carries its own binary, built from the same
  `make build` as the release archives.
- **Negative:** one bundle per notes root, since the extension name is fixed
  at build time.
- **Negative:** `--log-level` and `--git-author` are not exposed in the
  extension settings; they keep their defaults. A shared instance refuses a
  client with a different git author (ADR-0021), so a bundle and a
  hand-written entry on the same root must agree.
- **Risk / Note:** the binary is not signed. A bundle downloaded with a
  browser may carry the macOS quarantine attribute; a locally built bundle
  does not.

## References

- MCPB manifest specification: https://github.com/modelcontextprotocol/mcpb/blob/main/MANIFEST.md
- ADR-0012 (build targets), ADR-0021 (shared instance).
