#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

  # for shellcheck SC2154
  export output

  # Exported so helper functions running in subshells inherit it.
  export DODDER_BIN
}

teardown() {
  chflags_nouchg
}

# bats file_tags=user_story:mcp

# write_mcp_input writes a full JSON-RPC request stream (initialize,
# initialized notification, tools/list) to the given path.
function write_mcp_tools_list_input {
  local path="$1"
  {
    echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"0.1.0"}}}'
    echo '{"jsonrpc":"2.0","method":"notifications/initialized"}'
    echo '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}'
  } >"$path"
}

function mcp_initialize_no_workspace { # @test
  # Outside a workspace, the MCP server must still start and answer the
  # initialize call cleanly — otherwise every moxy/Claude Code session
  # launched from a non-workspace directory loses dodder entirely.
  # https://github.com/amarbel-llc/dodder/issues/116
  copy_from_version "$DIR"

  local input="$BATS_TEST_TMPDIR/mcp-init.jsonrpc"
  echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"0.1.0"}}}' >"$input"

  run timeout 5s "$DODDER_BIN" mcp <"$input"

  assert_success
  assert_output --regexp '"serverInfo":\{[^}]*"name":"dodder"'
  assert_output --regexp '"protocolVersion":"2024-11-05"'
}

function mcp_tools_list_no_workspace { # @test
  # Outside a workspace, workspace-scoped tools (status, checkin, diff,
  # read-checked_out) must not appear in tools/list. The grep is scoped
  # to the id:2 (tools/list) response so serverInfo's "name":"dodder"
  # from the initialize response doesn't pollute the list.
  # https://github.com/amarbel-llc/dodder/issues/116
  copy_from_version "$DIR"

  local input="$BATS_TEST_TMPDIR/mcp-tools-list.jsonrpc"
  write_mcp_tools_list_input "$input"

  run bash -o pipefail -c \
    'timeout 5s "'"$DODDER_BIN"'" mcp <"'"$input"'" | grep "\"id\":2" | grep -oE "\"name\":\"[a-z_-]+\"" | sort -u'

  assert_success
  assert_output - <<-EOM
		"name":"edit"
		"name":"format-blob"
		"name":"new"
		"name":"query"
		"name":"query-tag"
		"name":"query-type"
		"name":"reset-lock"
		"name":"show"
	EOM
}

function mcp_tools_list_with_workspace { # @test
  # Inside a workspace, the full tool set is advertised — including the
  # workspace-scoped tools skipped in the no-workspace case.
  # https://github.com/amarbel-llc/dodder/issues/116
  run_dodder_init_disable_age

  local input="$BATS_TEST_TMPDIR/mcp-tools-list.jsonrpc"
  write_mcp_tools_list_input "$input"

  run bash -o pipefail -c \
    'timeout 5s "'"$DODDER_BIN"'" mcp <"'"$input"'" | grep "\"id\":2" | grep -oE "\"name\":\"[a-z_-]+\"" | sort -u'

  assert_success
  assert_output - <<-EOM
		"name":"checkin"
		"name":"diff"
		"name":"edit"
		"name":"format-blob"
		"name":"new"
		"name":"query"
		"name":"query-tag"
		"name":"query-type"
		"name":"read-checked_out"
		"name":"reset-lock"
		"name":"show"
		"name":"status"
	EOM
}

# write_mcp_tool_call_input writes a JSON-RPC stream that invokes a
# tool by name, given a JSON arguments object. The third frame is the
# tools/call invocation; frames 1-2 are initialize + initialized.
function write_mcp_tool_call_input {
  local path="$1"
  local tool_name="$2"
  local arguments_json="$3"
  {
    echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"0.1.0"}}}'
    echo '{"jsonrpc":"2.0","method":"notifications/initialized"}'
    echo '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"'"$tool_name"'","arguments":'"$arguments_json"'}}'
  } >"$path"
}

# write_mcp_resource_read_input writes a JSON-RPC stream that reads a
# resource by URI. The third frame is the resources/read request; frames
# 1-2 are initialize + initialized.
function write_mcp_resource_read_input {
  local path="$1"
  local uri="$2"
  {
    echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"0.1.0"}}}'
    echo '{"jsonrpc":"2.0","method":"notifications/initialized"}'
    echo '{"jsonrpc":"2.0","id":2,"method":"resources/read","params":{"uri":"'"$uri"'"}}'
  } >"$path"
}

function mcp_show_type_object_has_non_empty_text { # @test
  # Strict MCP clients (e.g. Claude Code's zod validator) reject content
  # blocks whose `text` field is the empty string. `dodder show -format
  # log !md` produces empty stdout against a fresh repo's type, so
  # without a guard the MCP wrapper returns `"text":""` and the client
  # bails. Assert that the bridge falls back to a placeholder.
  # https://github.com/amarbel-llc/dodder/issues/213
  run_dodder_init_disable_age

  local input="$BATS_TEST_TMPDIR/mcp-show.jsonrpc"
  write_mcp_tool_call_input "$input" show '{"object_id":"!md"}'

  run bash -o pipefail -c \
    'timeout 5s "'"$DODDER_BIN"'" mcp <"'"$input"'" | grep "\"id\":2" | grep -oE "\"text\":\"[^\"]*\""'

  assert_success
  refute_output '"text":""'
}

function mcp_new_twice_does_not_leak_description { # @test
  # Two sequential `new` tool calls in one MCP session must not leak
  # flag state: Description.Set appends when a value is already
  # present, and the bridge reuses the registered command singletons,
  # so the second object's description came out concatenated with the
  # first's ("first description second description").
  # https://github.com/amarbel-llc/dodder/issues/247
  run_dodder_init_disable_age

  local input="$BATS_TEST_TMPDIR/mcp-new-twice.jsonrpc"
  {
    echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"0.1.0"}}}'
    echo '{"jsonrpc":"2.0","method":"notifications/initialized"}'
    echo '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"new","arguments":{"description":"first description"}}}'
    echo '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"new","arguments":{"description":"second description"}}}'
  } >"$input"

  run bash -o pipefail -c \
    'timeout 15s "'"$DODDER_BIN"'" mcp <"'"$input"'" | grep "\"id\":3"'

  assert_success
  assert_output --regexp '"second description"|second description'
  refute_output --regexp 'first description'
}

function mcp_locked_mutate_fails_unambiguously_and_reset_lock_recovers { # @test
  # A failed mutating operation intentionally leaves the env lock held
  # (see local_working_copy.Unlock); in the long-lived MCP server that
  # poisons every later mutate. The failure response must state the
  # lock state unambiguously and point at reset-lock, and reset-lock
  # must actually recover.
  # https://github.com/amarbel-llc/dodder/issues/249
  run_dodder_init_disable_age

  # Discover the env lock path from the repo's XDG state dir.
  run_dodder info xdg
  assert_success
  local state_home
  state_home="$(echo "$output" | grep -E 'STATE' | head -n1 | cut -d= -f2 | tr -d '"')"
  assert [ -n "$state_home" ]

  # Stage a stale lock file, as left behind by a dead process's failed
  # mutation.
  mkdir -p "$state_home"
  touch "$state_home/lock"

  local input="$BATS_TEST_TMPDIR/mcp-locked-new.jsonrpc"
  write_mcp_tool_call_input "$input" new '{"description":"blocked by lock"}'

  run bash -o pipefail -c \
    'timeout 15s "'"$DODDER_BIN"'" mcp <"'"$input"'" | grep "\"id\":2"'
  assert_success
  assert_output --regexp 'REPO LOCKED'
  assert_output --regexp 'reset-lock'

  local reset_input="$BATS_TEST_TMPDIR/mcp-reset-lock.jsonrpc"
  write_mcp_tool_call_input "$reset_input" reset-lock '{}'

  run bash -o pipefail -c \
    'timeout 15s "'"$DODDER_BIN"'" mcp <"'"$reset_input"'" | grep "\"id\":2"'
  assert_success
  assert_output --regexp 'no longer locked'
  assert [ ! -e "$state_home/lock" ]

  local after_input="$BATS_TEST_TMPDIR/mcp-new-after-reset.jsonrpc"
  write_mcp_tool_call_input "$after_input" new '{"description":"after reset"}'

  run bash -o pipefail -c \
    'timeout 15s "'"$DODDER_BIN"'" mcp <"'"$after_input"'" | grep "\"id\":2"'
  assert_success
  assert_output --regexp 'after reset'
  refute_output --regexp 'REPO LOCKED'
}

# bats test_tags=repo_id
function mcp_repo_id_routes_tool_to_repo { # @test
  # FDR-0019 Phase A: a bridge-routed tool's optional repo_id param selects
  # which repo the call opens. Targeting the server's own cwd repo (.default)
  # returns its zettel; targeting a different, nonexistent repo errors
  # (proving the param changes the open target, not just decoration); an
  # unwired scope (//system) is rejected by repo_id.CheckSupported.
  run_dodder_init_disable_age

  to_add="$(mktemp)"
  {
    echo "---"
    echo "# repo_id routing probe"
    echo "- task"
    echo "! md"
    echo "---"
  } >"$to_add"
  run_dodder new -edit=false "$to_add"
  assert_success

  # .default -> the server's repo -> query returns the zettel
  local in_default="$BATS_TEST_TMPDIR/mcp-q-default.jsonrpc"
  write_mcp_tool_call_input "$in_default" query '{"query":[":z"],"repo_id":".default"}'
  run bash -o pipefail -c \
    'timeout 5s "'"$DODDER_BIN"'" mcp <"'"$in_default"'" | grep "\"id\":2"'
  assert_success
  assert_output --regexp 'repo_id routing probe'

  # nonexistent user-scope repo -> errors (different open target)
  local in_missing="$BATS_TEST_TMPDIR/mcp-q-missing.jsonrpc"
  write_mcp_tool_call_input "$in_missing" query '{"query":[":z"],"repo_id":"nonexistent"}'
  run bash -o pipefail -c \
    'timeout 5s "'"$DODDER_BIN"'" mcp <"'"$in_missing"'" | grep "\"id\":2"'
  assert_success
  assert_output --regexp 'not in a dodder directory'
  # the error names repos/nonexistent/ — proof the param routed the open
  assert_output --regexp 'repos/nonexistent'
  refute_output --regexp 'repo_id routing probe'

  # //system is parsed but not yet resolvable -> CheckSupported reject
  local in_system="$BATS_TEST_TMPDIR/mcp-q-system.jsonrpc"
  write_mcp_tool_call_input "$in_system" query '{"query":[":z"],"repo_id":"//backup"}'
  run bash -o pipefail -c \
    'timeout 5s "'"$DODDER_BIN"'" mcp <"'"$in_system"'" | grep "\"id\":2"'
  assert_success
  assert_output --regexp 'system scope is not yet resolvable'
}

# bats test_tags=repo_id
function mcp_repo_scoped_resources_route_to_repo { # @test
  # FDR-0019 Phase B: the resource surface is repo-scoped. Reading
  # dodder:///repos lists the repos in scope; a repo-scoped
  # dodder:///repos/.default/objects routes the read to that repo and
  # returns its objects; the legacy un-segmented dodder://objects still
  # resolves to the auto/default repo (CWD-auto sugar); a repo-scoped
  # read of a nonexistent repo surfaces the open failure as resource
  # content. https://github.com/amarbel-llc/dodder/issues/275
  run_dodder_init_disable_age

  to_add="$(mktemp)"
  {
    echo "---"
    echo "# repo scoped resource probe"
    echo "- task"
    echo "! md"
    echo "---"
  } >"$to_add"
  run_dodder new -edit=false "$to_add"
  assert_success

  # dodder:///repos -> lists at least one repo
  local in_repos="$BATS_TEST_TMPDIR/mcp-res-repos.jsonrpc"
  write_mcp_resource_read_input "$in_repos" 'dodder:///repos'
  run bash -o pipefail -c \
    'timeout 5s "'"$DODDER_BIN"'" mcp <"'"$in_repos"'" | grep "\"id\":2"'
  assert_success
  assert_output --regexp 'total_repos[^0-9]+[1-9]'

  # repo-scoped objects listing routes to the cwd repo -> returns the zettel
  local in_scoped="$BATS_TEST_TMPDIR/mcp-res-scoped.jsonrpc"
  write_mcp_resource_read_input "$in_scoped" 'dodder:///repos/.default/objects'
  run bash -o pipefail -c \
    'timeout 5s "'"$DODDER_BIN"'" mcp <"'"$in_scoped"'" | grep "\"id\":2"'
  assert_success
  assert_output --regexp 'repo scoped resource probe'

  # legacy un-segmented form still resolves (CWD-auto sugar)
  local in_legacy="$BATS_TEST_TMPDIR/mcp-res-legacy.jsonrpc"
  write_mcp_resource_read_input "$in_legacy" 'dodder://objects'
  run bash -o pipefail -c \
    'timeout 5s "'"$DODDER_BIN"'" mcp <"'"$in_legacy"'" | grep "\"id\":2"'
  assert_success
  assert_output --regexp 'repo scoped resource probe'

  # repo-scoped read of a nonexistent repo -> open failure as content
  local in_missing="$BATS_TEST_TMPDIR/mcp-res-missing.jsonrpc"
  write_mcp_resource_read_input "$in_missing" 'dodder:///repos/nonexistent/objects'
  run bash -o pipefail -c \
    'timeout 5s "'"$DODDER_BIN"'" mcp <"'"$in_missing"'" | grep "\"id\":2"'
  assert_success
  assert_output --regexp 'not in a dodder directory|repos/nonexistent'
  refute_output --regexp 'repo scoped resource probe'
}

function mcp_query_empty_result_has_non_empty_text { # @test
  # Same root cause as the show-type test: an empty result set yields
  # empty stdout and an empty content block; assert the placeholder.
  # https://github.com/amarbel-llc/dodder/issues/213
  run_dodder_init_disable_age

  local input="$BATS_TEST_TMPDIR/mcp-query-empty.jsonrpc"
  write_mcp_tool_call_input "$input" query '{"query":[":z"]}'

  run bash -o pipefail -c \
    'timeout 5s "'"$DODDER_BIN"'" mcp <"'"$input"'" | grep "\"id\":2" | grep -oE "\"text\":\"[^\"]*\""'

  assert_success
  refute_output '"text":""'
}
