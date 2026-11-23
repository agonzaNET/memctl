#!/usr/bin/env bash
#
# mcpb-build.test.sh — regression test for mcpb-build.sh.
#
# Runs mcpb-build.sh with a stub in place of the MCPB CLI (no Node.js, no
# network) and checks what it MUST do:
#
#   bad arguments           missing or malformed version or name        → 2
#   staged manifest         name, display_name and version set           → 0
#   default name            NAME memctl gives display_name "memctl"      → 0
#   arguments               command and --root ${user_config.root} kept  → 0
#   scope                   optional string, default "", as MEMCTL_SCOPE → 0
#   binary                  server/memctl present and executable         → 0
#   output                  OUTDIR/NAME_X.Y.Z_darwin_arm64.mcpb written  → 0
#
# Run directly: scripts/mcpb-build.test.sh (exits 0 on success, 1 on failure).

set -u

script_dir=$(cd "$(dirname "$0")" && pwd)
target="$script_dir/mcpb-build.sh"
[ -f "$target" ] || { echo "FAIL: cannot find $target"; exit 1; }

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

tests_run=0
tests_failed=0

check() { # description, condition (evaluated)
	tests_run=$((tests_run + 1))
	if eval "$2"; then
		echo "ok   $1"
	else
		echo "FAIL $1"
		tests_failed=$((tests_failed + 1))
	fi
}

# The stub copies the staged bundle aside and writes the output file.
stub="$tmp/mcpb-stub"
cat > "$stub" <<STUB
#!/usr/bin/env bash
[ "\$1" = pack ] || exit 9
rm -rf "$tmp/seen" && cp -Rp "\$2" "$tmp/seen" && echo bundle > "\$3"
STUB
chmod +x "$stub"

for args in "" "1.2 docs $tmp/out" "1.2.3 Docs $tmp/out" "1.2.3 docs" "x1.2.3 docs $tmp/out"; do
	# shellcheck disable=SC2086
	MCPB="$stub" bash "$target" $args >/dev/null 2>&1
	check "bad arguments '$args' exit 2" "[ $? -eq 2 ]"
done

MCPB="$stub" bash "$target" 1.2.3 infradocs "$tmp/out" >"$tmp/log" 2>&1
check "named build exits 0" "[ $? -eq 0 ]"
m="$tmp/seen/manifest.json"
check "name set" "[ \"\$(jq -r .name '$m')\" = infradocs ]"
check "display_name set" "[ \"\$(jq -r .display_name '$m')\" = 'memctl (infradocs)' ]"
check "version set" "[ \"\$(jq -r .version '$m')\" = 1.2.3 ]"
check "command kept" "[ \"\$(jq -r .server.mcp_config.command '$m')\" = '\${__dirname}/server/memctl' ]"
check "args kept" "[ \"\$(jq -c .server.mcp_config.args '$m')\" = '[\"mcp\",\"--root\",\"\${user_config.root}\"]' ]"
check "root is a required directory" "[ \"\$(jq -c '.user_config.root | [.type, .required]' '$m')\" = '[\"directory\",true]' ]"
check "scope is an optional string with default empty" "[ \"\$(jq -c '.user_config.scope | [.type, .required, .default]' '$m')\" = '[\"string\",false,\"\"]' ]"
check "scope passed as MEMCTL_SCOPE" "[ \"\$(jq -r .server.mcp_config.env.MEMCTL_SCOPE '$m')\" = '\${user_config.scope}' ]"
check "binary executable" "[ -x '$tmp/seen/server/memctl' ]"
check "LICENSE and README.md staged" "[ -f '$tmp/seen/LICENSE' ] && [ -f '$tmp/seen/README.md' ]"
check "output written" "[ -f '$tmp/out/infradocs_1.2.3_darwin_arm64.mcpb' ]"

MCPB="$stub" bash "$target" 1.2.3 memctl "$tmp/out" >/dev/null 2>&1
check "default name build exits 0" "[ $? -eq 0 ]"
check "default display_name" "[ \"\$(jq -r .display_name '$m')\" = memctl ]"

echo "$tests_run tests, $tests_failed failed"
[ "$tests_failed" -eq 0 ]
