#!/usr/bin/env bash
#
# release-notes.test.sh — regression test for release-notes.sh.
#
# Each case gives a CHANGELOG, a version, and what release-notes.sh MUST do:
#
#   middle section          the body between its heading and the next one → 0
#   last section            stops before the link references             → 0
#   missing version         no "## [X.Y.Z]" heading                       → 1
#   empty section           a heading with no entries                     → 1
#   prefix of a version     "0.1" is a prefix of 0.1.0 and 0.1.01         → 1
#   unreleased only         "## [Unreleased]" is never a version          → 1
#
# Run directly: scripts/release-notes.test.sh (exits 0 on success, 1 on failure).

set -u

script_dir=$(cd "$(dirname "$0")" && pwd)
target="$script_dir/release-notes.sh"
[ -f "$target" ] || { echo "FAIL: cannot find $target"; exit 1; }

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

tests_run=0
tests_failed=0

changelog="$tmp/CHANGELOG.md"
cat > "$changelog" <<'MD'
# Changelog

Intro text.

## [Unreleased]

### Added

- Something new.

## [0.2.0] - 2027-01-10

### Added

- Second feature.

### Fixed

- A fix.

## [0.1.01] - 2026-12-02

## [0.1.0] - 2026-12-01

### Added

- First release.

[Unreleased]: https://example.org/compare/v0.2.0...HEAD
[0.2.0]: https://example.org/compare/v0.1.0...v0.2.0
MD

# expect NAME VERSION STATUS [EXPECTED_STDOUT]
expect() {
	local name=$1 version=$2 want_status=$3 want_out=${4-}
	tests_run=$((tests_run + 1))
	local out status
	out=$(bash "$target" "$version" "$changelog" 2>/dev/null)
	status=$?
	if [ "$status" -ne "$want_status" ]; then
		echo "FAIL: $name: exit $status, want $want_status"
		tests_failed=$((tests_failed + 1))
		return
	fi
	if [ "$want_status" -eq 0 ] && [ "$out" != "$want_out" ]; then
		echo "FAIL: $name: output"
		printf '  got:\n%s\n  want:\n%s\n' "$out" "$want_out"
		tests_failed=$((tests_failed + 1))
		return
	fi
	echo "ok:   $name"
}

expect "middle section" 0.2.0 0 "### Added

- Second feature.

### Fixed

- A fix."
expect "last section" 0.1.0 0 "### Added

- First release."
expect "missing version" 0.3.0 1
expect "empty section" 0.1.01 1
expect "prefix of a version" 0.1 1
expect "unreleased only" Unreleased 1

echo
echo "$tests_run run, $tests_failed failed"
[ "$tests_failed" -eq 0 ]
