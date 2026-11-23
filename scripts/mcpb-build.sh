#!/usr/bin/env bash
#
# mcpb-build.sh — build a Claude Desktop extension (MCP Bundle) of memctl.
#
#   scripts/mcpb-build.sh X.Y.Z NAME OUTDIR
#
# Builds the darwin/arm64 binary with `make build` and VERSION=X.Y.Z, stages
# it as server/memctl with LICENSE, README.md and mcpb/manifest.json (name,
# display_name and version set from NAME and X.Y.Z), and packs
# OUTDIR/NAME_X.Y.Z_darwin_arm64.mcpb. NAME is the server name Claude Desktop
# shows; build one bundle per notes root. See ADR-0024.
#
# The MCPB CLI command comes from $MCPB (default: the pinned CLI through npx).
# It is called as: $MCPB pack STAGEDIR OUTFILE

set -euo pipefail

version=${1:-}
name=${2:-}
out=${3:-}
mcpb=${MCPB:-npx --yes @anthropic-ai/mcpb@2.1.2}

if ! [[ $version =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$ ]] ||
	! [[ $name =~ ^[a-z0-9][a-z0-9-]*$ ]] || [ -z "$out" ]; then
	echo "usage: mcpb-build.sh X.Y.Z NAME OUTDIR   (NAME: lowercase letters, digits, hyphens)" >&2
	exit 2
fi
command -v jq >/dev/null || { echo "mcpb-build: jq is required" >&2; exit 1; }

repo=$(cd "$(dirname "$0")/.." && pwd)
mkdir -p "$out"
out=$(cd "$out" && pwd)
stage=$(mktemp -d)
trap 'rm -rf "$stage"' EXIT
bundle="$stage/bundle"
mkdir -p "$bundle/server"

make -C "$repo" --no-print-directory build GOOS=darwin GOARCH=arm64 VERSION="$version" BIN_DIR="$bundle/server"
chmod 0755 "$bundle/server/memctl"
cp "$repo/LICENSE" "$repo/README.md" "$bundle/"
chmod 0644 "$bundle/LICENSE" "$bundle/README.md"
jq --arg name "$name" --arg version "$version" \
	'.name = $name | .display_name = (if $name == "memctl" then "memctl" else "memctl (" + $name + ")" end) | .version = $version' \
	"$repo/mcpb/manifest.json" > "$bundle/manifest.json"

file="$out/${name}_${version}_darwin_arm64.mcpb"
rm -f "$file"
$mcpb pack "$bundle" "$file"
[ -f "$file" ] || { echo "mcpb-build: $file was not written" >&2; exit 1; }
echo "mcpb-build: $file"
