#!/usr/bin/env bash
#
# release-notes.sh — print the CHANGELOG section of one release.
#
#   scripts/release-notes.sh X.Y.Z [CHANGELOG.md]
#
# Prints the body of the "## [X.Y.Z] - YYYY-MM-DD" section, without its
# heading, up to the next "## " heading or the link references at the end,
# with leading and trailing blank lines removed. Exits 1 when the section is
# missing or empty, so a release cannot ship without notes (ADR-0007, stage
# plan 0003). Used by the release workflow.

set -euo pipefail

version=${1:-}
changelog=${2:-CHANGELOG.md}

if ! [[ $version =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
	echo "release-notes: the version must be X.Y.Z, got '$version'" >&2
	exit 1
fi

body=$(awk -v heading="## [$version] - " '
	index($0, heading) == 1 { found = 1; next }
	found && (/^## / || /^\[[^]]+\]: /) { exit }
	found { print }
' "$changelog" | sed -e '/./,$!d' | awk '
	{ lines[NR] = $0 }
	END { last = NR; while (last > 0 && lines[last] ~ /^[[:space:]]*$/) last--; for (i = 1; i <= last; i++) print lines[i] }
')

if [ -z "$body" ]; then
	echo "release-notes: $changelog has no entries for $version under '## [$version] - <date>'" >&2
	exit 1
fi
printf '%s\n' "$body"
