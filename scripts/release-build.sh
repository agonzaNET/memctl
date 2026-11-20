#!/usr/bin/env bash
#
# release-build.sh — build the release archives of memctl.
#
#   scripts/release-build.sh X.Y.Z OUTDIR
#
# For each release target of ADR-0012 (linux/amd64, linux/arm64,
# darwin/arm64), runs `make build` with VERSION=X.Y.Z and packs
# memctl_X.Y.Z_<os>_<arch>.tar.gz holding memctl, LICENSE and README.md.
# Writes SHA256SUMS over the archives. Used by the release workflow and by
# `make dist`; see stage plan 0003.

set -euo pipefail

version=${1:-}
out=${2:-}
targets=("linux/amd64" "linux/arm64" "darwin/arm64")

if ! [[ $version =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || [ -z "$out" ]; then
	echo "usage: release-build.sh X.Y.Z OUTDIR" >&2
	exit 2
fi

repo=$(cd "$(dirname "$0")/.." && pwd)
mkdir -p "$out"
out=$(cd "$out" && pwd)
stage=$(mktemp -d)
trap 'rm -rf "$stage"' EXIT

# Archive members owned by root rather than the build user, where tar can.
owner=()
if tar --version 2>/dev/null | grep -q 'GNU tar'; then
	owner=(--owner=0 --group=0 --numeric-owner)
fi

archives=()
for t in "${targets[@]}"; do
	goos=${t%/*}
	goarch=${t#*/}
	name="memctl_${version}_${goos}_${goarch}"
	make -C "$repo" --no-print-directory build GOOS="$goos" GOARCH="$goarch" VERSION="$version" BIN_DIR="$stage/$name"
	cp "$repo/LICENSE" "$repo/README.md" "$stage/$name/"
	chmod 0755 "$stage/$name/memctl"
	chmod 0644 "$stage/$name/LICENSE" "$stage/$name/README.md"
	tar -C "$stage/$name" "${owner[@]}" -czf "$out/$name.tar.gz" memctl LICENSE README.md
	archives+=("$name.tar.gz")
done

if command -v sha256sum >/dev/null; then
	(cd "$out" && sha256sum "${archives[@]}" > SHA256SUMS)
else
	(cd "$out" && shasum -a 256 "${archives[@]}" > SHA256SUMS)
fi
echo "release-build: ${#archives[@]} archives and SHA256SUMS in $out"
