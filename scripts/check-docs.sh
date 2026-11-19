#!/usr/bin/env bash
#
# check-docs.sh — methodology consistency check.
#
# Asserts that the hand-maintained index tables agree with the files they
# summarise, and that the agent-facing tool text names no internal reference:
#
#   1. Every docs/specs/NNNN-slug/spec.md Status matches its row in
#      docs/specs/README.md (and each appears in the other).
#   2. Every docs/adr/NNNN-*.md Status and Area match its row in
#      docs/adr/README.md (and each appears in the other) (ADR-0009).
#   3. Spec/ADR numbering gaps are reported as warnings (informational only).
#   4. Every spec's "**Governing ADRs:**" line (which may wrap across several
#      lines) names the same set of ADRs as that spec's row in
#      docs/specs/README.md.
#   5. The agent-facing text of the MCP tools names no internal reference
#      (ADR-0013): every non-comment line of internal/mcpserver/tools.go between
#      the markers `// agent-facing text: begin` and `// agent-facing text: end`
#      has no ADR or spec number in any letter case, with or without a hyphen
#      or space (ADR-0013, adr 13, SPEC-0001), no requirement number (R12, R8a),
#      no "design decision", no repository directory (docs/, internal/, cmd/,
#      scripts/) and no .go or .py file name. Note paths and .md file names are
#      allowed: agents pass them to the tools. Skipped while tools.go does not
#      exist; once it exists, missing markers fail the check, so a reformatted
#      marker cannot turn the scan off. Project-history wording ("no longer",
#      "used to") is not machine-checked; review it by hand.
#
# Run from anywhere in the repository. Exits non-zero on any real mismatch
# (1 + 2 + 4 + 5); warnings (3) do not affect the exit code. Run by
# `make verify` in Phase 4 of the workflow.
#
# Portable: POSIX-ish bash, no associative arrays (works on macOS bash 3.2).
#
# Tested by scripts/check-docs.test.sh — a fixtures-based regression test that
# asserts this script passes on a consistent docs tree and fails on a drifted
# one. Run it after editing this script (see Phase 4 in docs/agents/workflow.md).

set -u

repo_root=$(cd "$(dirname "$0")/.." && pwd)
cd "$repo_root" || exit 1

fail=0

# First non-blank line after a "## Status" header, trimmed.
status_of() {
  awk '
    seen && NF { sub(/^[[:space:]]+/, ""); sub(/[[:space:]]+$/, ""); print; exit }
    /^##+[[:space:]]+Status[[:space:]]*$/ { seen = 1 }
  ' "$1"
}

# Value of the "**Area:**" metadata line, trimmed (ADRs only).
area_of() {
  awk '
    /^[[:space:]]*\*\*Area:\*\*/ {
      sub(/^[[:space:]]*\*\*Area:\*\*[[:space:]]*/, "")
      sub(/[[:space:]]+$/, "")
      print; exit
    }
  ' "$1"
}

# ---- 1. Specs: files vs docs/specs/README.md index -------------------------

specs_from_files() {
  for d in docs/specs/*/; do
    name=$(basename "$d")
    [ -f "$d/spec.md" ] || continue
    num=${name%%-*}
    printf 'SPEC-%s | %s\n' "$num" "$(status_of "$d/spec.md")"
  done | sort
}

specs_from_index() {
  awk -F'|' '
    /^\|[[:space:]]*SPEC-[0-9]/ {
      id = $2; st = $4
      gsub(/^[[:space:]]+|[[:space:]]+$/, "", id)
      gsub(/^[[:space:]]+|[[:space:]]+$/, "", st)
      print id " | " st
    }
  ' docs/specs/README.md | sort
}

echo "==> Specs: spec.md Status vs specs/README.md index"
if diff_out=$(diff <(specs_from_files) <(specs_from_index)); then
  echo "    OK"
else
  echo "    MISMATCH (< spec file, > index row):"
  echo "$diff_out" | sed 's/^/    /'
  fail=1
fi

# ---- 2. ADRs: files vs docs/adr/README.md index ----------------------------

adrs_from_files() {
  for f in docs/adr/[0-9]*.md; do
    [ -f "$f" ] || continue
    base=$(basename "$f")
    num=${base%%-*}
    printf 'ADR-%s | %s | %s\n' "$num" "$(status_of "$f")" "$(area_of "$f")"
  done | sort
}

# Index rows only: lines that start with "| [ADR-NNNN]". This deliberately
# excludes prose ADR links (e.g. the foundational-ADR callout), which do not
# begin with a table pipe. Columns: | ID | Title | Status | Area |.
adrs_from_index() {
  awk -F'|' '
    /^\|[[:space:]]*\[ADR-[0-9]/ {
      match($2, /ADR-[0-9]+/); id = substr($2, RSTART, RLENGTH)
      st = $4; ar = $5
      gsub(/^[[:space:]]+|[[:space:]]+$/, "", st)
      gsub(/^[[:space:]]+|[[:space:]]+$/, "", ar)
      print id " | " st " | " ar
    }
  ' docs/adr/README.md | sort
}

echo "==> ADRs: ADR file Status + Area vs adr/README.md index"
if diff_out=$(diff <(adrs_from_files) <(adrs_from_index)); then
  echo "    OK"
else
  echo "    MISMATCH (< adr file, > index row):"
  echo "$diff_out" | sed 's/^/    /'
  fail=1
fi

# ---- 3. Numbering gaps (warnings only) -------------------------------------

# Retired IDs: never reused, not reported. Space-separated four-digit IDs.
RETIRED_SPECS=""
RETIRED_ADRS=""

report_gaps() {
  label=$1; retired=$2; shift 2
  prev=""
  for n in "$@"; do
    if [ -n "$prev" ]; then
      i=$((10#$prev + 1))
      while [ "$i" -lt "$((10#$n))" ]; do
        id=$(printf '%04d' "$i")
        case " $retired " in
          *" $id "*) ;;
          *) printf '    WARNING: %s number %s is missing (gap between %s and %s)\n' \
               "$label" "$id" "$prev" "$n" ;;
        esac
        i=$((i + 1))
      done
    fi
    prev=$n
  done
}

echo "==> Numbering gaps (informational)"
spec_nums=$(for d in docs/specs/*/; do
  name=$(basename "$d")
  [ -f "$d/spec.md" ] && echo "${name%%-*}"
done | sort -u)
adr_nums=$(for f in docs/adr/[0-9]*.md; do basename "$f" | sed 's/-.*//'; done | sort -u)
report_gaps "SPEC" "$RETIRED_SPECS" $spec_nums
report_gaps "ADR" "$RETIRED_ADRS" $adr_nums
echo "    (gaps are informational; they do not fail the check)"

# ---- 4. Specs: "Governing ADRs" line matches the index row -----------------

# All ADR-NNNN tokens on a spec's "**Governing ADRs:**" line, sorted and
# de-duplicated. The line may wrap across several physical lines, so collection
# starts at the marker and continues until the first blank line.
spec_governing_adrs() {
  awk '
    /\*\*Governing ADRs:\*\*/ { collecting = 1 }
    collecting {
      if ($0 ~ /^[[:space:]]*$/) exit
      buf = buf " " $0
    }
    END { print buf }
  ' "$1" | grep -oE 'ADR-[0-9]+' | sort -u | tr '\n' ' '
}

# The ADR-NNNN tokens in a given spec's row of docs/specs/README.md, sorted/unique.
index_governing_adrs() {
  grep -E "^\|[[:space:]]*$1[[:space:]]*\|" docs/specs/README.md \
    | grep -oE 'ADR-[0-9]+' | sort -u | tr '\n' ' '
}

echo "==> Specs: spec.md Governing ADRs vs specs/README.md index"
gov_ok=1
for d in docs/specs/*/; do
  name=$(basename "$d")
  [ -f "$d/spec.md" ] || continue
  id="SPEC-${name%%-*}"
  file_set=$(spec_governing_adrs "$d/spec.md")
  index_set=$(index_governing_adrs "$id")
  if [ "$file_set" != "$index_set" ]; then
    echo "    MISMATCH $id (< spec file, > index row):"
    echo "        spec : $file_set"
    echo "        index: $index_set"
    gov_ok=0
    fail=1
  fi
done
[ "$gov_ok" -eq 1 ] && echo "    OK"

# ---- 5. Agent-facing tool text: no internal references --------------------
#
# The MCP tool descriptions and schemas are read by AI agents first (ADR-0013).
# An ADR or spec number, a requirement number or a repository path tells an
# agent nothing it can act on, so none may appear in the text it reads: every
# non-comment line of internal/mcpserver/tools.go between its two markers. Go
# comment lines are for maintainers and may name ADRs. Go code outside the
# markers is not agent-facing text and is not scanned.
#
# The path alternative skips a segment that follows ".", "/" or "-", so a note
# path such as ops/internal/x.md is not read as the internal/ tree. .md is not
# in the file-name alternative: note paths are agent-facing on purpose.

internal_ref_re='[Aa][Dd][Rr][- ]?[0-9]|[Ss][Pp][Ee][Cc][- ]?[0-9]|[Dd]esign decision|(^|[^A-Za-z0-9_./-])(docs|internal|cmd|scripts)/[A-Za-z0-9_.-]|[A-Za-z0-9_-]+\.(go|py)([^A-Za-z0-9_]|$)|(^|[^A-Za-z0-9_])R[0-9]+[a-z]?([^A-Za-z0-9_]|$)'

tools_go=internal/mcpserver/tools.go
tools_begin='// agent-facing text: begin'
tools_end='// agent-facing text: end'

echo "==> Agent-facing text: no ADR/spec/requirement references in the MCP tools"
if [ ! -f "$tools_go" ]; then
  echo "    SKIP ($tools_go not found yet)"
elif ! grep -qxF "$tools_begin" "$tools_go" || ! grep -qxF "$tools_end" "$tools_go"; then
  echo "    MISMATCH: $tools_go has no 'agent-facing text: begin' and 'agent-facing text: end' markers, so the tool text cannot be found and scanned"
  fail=1
else
  # Print every scanned line and a blank line for every other line, so grep -n
  # reports real line numbers and the pattern sees only the line's own text.
  hits=$(awk -v b="$tools_begin" -v e="$tools_end" '
    $0 == b { on = 1; print ""; next }
    $0 == e { on = 0; print ""; next }
    { if (on && !/^[[:space:]]*\/\//) print; else print "" }
  ' "$tools_go" | grep -nE "$internal_ref_re" \
    | sed 's/^/        internal\/mcpserver\/tools.go:/')
  if [ -z "$hits" ]; then
    echo "    OK"
  else
    echo "    MISMATCH: agent-facing text names an internal reference (ADR-0013):"
    printf '%s\n' "$hits"
    echo "    — say what the agent needs instead (see docs/agents/tools.md)"
    fail=1
  fi
fi

# ---- Result ----------------------------------------------------------------

echo
if [ "$fail" -eq 0 ]; then
  echo "check-docs: PASS"
else
  echo "check-docs: FAIL — fix the mismatches above so the indexes and the tool text match the rules."
fi
exit "$fail"
