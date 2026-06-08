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
