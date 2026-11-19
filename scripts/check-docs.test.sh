#!/usr/bin/env bash
#
# check-docs.test.sh — regression test for check-docs.sh.
#
# Builds a self-contained fixture tree in a temp dir, copies the real
# check-docs.sh into it, and runs it against variants. Each case states the
# exit status check-docs.sh MUST have:
#
#   clean                   consistent indexes, no tools.go yet          → 0
#   spec status drift       index row Status differs from spec.md       → 1
#   adr area drift          index row Area differs from the ADR file    → 1
#   governing adrs drift    spec's Governing ADRs differ from its row   → 1
#   tools clean             markers, a note path ending in .md          → 0
#   tools adr reference     "see ADR-0001" between the markers          → 1
#   tools comment reference "ADR-0001" in a Go comment between markers  → 0
#   tools outside markers   "ADR-0001" in code outside the markers      → 0
#   tools requirement       "R12" between the markers                   → 1
#   tools repo path         "internal/index" between the markers        → 1
#   tools go file           "tools.go" between the markers              → 1
#   tools no markers        tools.go exists without the markers         → 1
#
# This pins the property that matters for a drift gate: it passes when the
# docs agree and fails when they don't, guarding against the silent false
# negative (PASS while the docs have drifted).
#
# Run directly: scripts/check-docs.test.sh (exits 0 on success, 1 on failure).

set -u

script_dir=$(cd "$(dirname "$0")" && pwd)
target="$script_dir/check-docs.sh"
[ -f "$target" ] || { echo "FAIL: cannot find $target"; exit 1; }

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

tests_run=0
tests_failed=0

# build_fixture DIR — a tree every check passes against.
build_fixture() {
  root=$1
  mkdir -p "$root/scripts" "$root/docs/specs/0001-foo" "$root/docs/adr"
  cp "$target" "$root/scripts/check-docs.sh"

  cat > "$root/docs/specs/0001-foo/spec.md" <<'X'
# SPEC-0001 - Foo

## Status

Implemented

**Governing ADRs:** ADR-0001,
ADR-0002

## Purpose

Foo.
X
  cat > "$root/docs/specs/README.md" <<'X'
# Specs

| ID | Title | Status | Governing ADRs |
|---|---|---|---|
| SPEC-0001 | Foo | Implemented | ADR-0001, ADR-0002 |
X
  for n in 0001 0002; do
    cat > "$root/docs/adr/$n-bar.md" <<X
# ADR-$n - Bar

## Status

Accepted

**Area:** Methodology

## Context

Why.
X
  done
  cat > "$root/docs/adr/README.md" <<'X'
# ADRs

See [ADR-0001](0001-bar.md) for the foundational set.

| ID | Title | Status | Area |
|---|---|---|---|
| [ADR-0001](0001-bar.md) | Bar | Accepted | Methodology |
| [ADR-0002](0002-bar.md) | Bar | Accepted | Methodology |
X
}

# write_tools DIR LINE — tools.go with LINE between the markers.
write_tools() {
  mkdir -p "$1/internal/mcpserver"
  cat > "$1/internal/mcpserver/tools.go" <<X
package mcpserver

// Maintainer note outside the markers may name ADR-0013.
const outside = "ADR-0001 outside the markers"

// agent-facing text: begin
const descRead = "Read one note by title, alias or relative path such as ops/pa-850.md."
$2
// agent-facing text: end
X
}

# expect NAME WANT_RC — run check-docs.sh in $tmp/NAME and compare the status.
expect() {
  name=$1; want=$2
  tests_run=$((tests_run + 1))
  out=$(bash "$tmp/$name/scripts/check-docs.sh" 2>&1); rc=$?
  [ "$rc" -ne 0 ] && rc=1
  if [ "$rc" -eq "$want" ]; then
    echo "ok   $name"
  else
    echo "FAIL $name: want exit $want, got $rc"
    echo "$out" | sed 's/^/     /'
    tests_failed=$((tests_failed + 1))
  fi
}

case_dir() { d="$tmp/$1"; build_fixture "$d"; }

case_dir clean;                expect clean 0

case_dir spec_status_drift
sed -i.bak 's/| Foo | Implemented |/| Foo | Accepted |/' "$d/docs/specs/README.md"
expect spec_status_drift 1

case_dir adr_area_drift
sed -i.bak 's/| \[ADR-0002\](0002-bar.md) | Bar | Accepted | Methodology |/| [ADR-0002](0002-bar.md) | Bar | Accepted | Tools |/' "$d/docs/adr/README.md"
expect adr_area_drift 1

case_dir governing_adrs_drift
sed -i.bak 's/| ADR-0001, ADR-0002 |/| ADR-0001 |/' "$d/docs/specs/README.md"
expect governing_adrs_drift 1

case_dir tools_clean;          write_tools "$d" 'const descEdit = "Edit a note; read it first to get its sha."'
expect tools_clean 0

case_dir tools_adr_reference;  write_tools "$d" 'const descX = "Fails as described, see ADR-0001."'
expect tools_adr_reference 1

case_dir tools_comment_reference; write_tools "$d" '// Governed by ADR-0001 and SPEC-0001 R3.'
expect tools_comment_reference 0

case_dir tools_outside_markers; write_tools "$d" ''
expect tools_outside_markers 0

case_dir tools_requirement;    write_tools "$d" 'const descX = "Required by R12."'
expect tools_requirement 1

case_dir tools_repo_path;      write_tools "$d" 'const descX = "Ranked by internal/index."'
expect tools_repo_path 1

case_dir tools_go_file;        write_tools "$d" 'const descX = "Defined in tools.go."'
expect tools_go_file 1

case_dir tools_no_markers;     write_tools "$d" ''
sed -i.bak '/agent-facing text/d' "$d/internal/mcpserver/tools.go"
expect tools_no_markers 1

echo
if [ "$tests_failed" -eq 0 ]; then
  echo "check-docs.test: PASS ($tests_run cases)"
  exit 0
fi
echo "check-docs.test: FAIL ($tests_failed of $tests_run cases)"
exit 1
