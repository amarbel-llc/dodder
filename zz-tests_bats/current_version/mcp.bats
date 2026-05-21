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
  # read_checked_out) must not appear in tools/list.
  # https://github.com/amarbel-llc/dodder/issues/116
  copy_from_version "$DIR"

  local input="$BATS_TEST_TMPDIR/mcp-tools-list.jsonrpc"
  write_mcp_tools_list_input "$input"

  run bash -o pipefail -c \
    'timeout 5s "'"$DODDER_BIN"'" mcp <"'"$input"'" | grep -oE "\"name\":\"dodder_[a-z_]+\"" | sort -u'

  assert_success
  assert_output - <<-EOM
		"name":"dodder_edit"
		"name":"dodder_format_blob"
		"name":"dodder_new"
		"name":"dodder_query"
		"name":"dodder_show"
		"name":"dodder_tag_query"
		"name":"dodder_type_query"
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
    'timeout 5s "'"$DODDER_BIN"'" mcp <"'"$input"'" | grep -oE "\"name\":\"dodder_[a-z_]+\"" | sort -u'

  assert_success
  assert_output - <<-EOM
		"name":"dodder_checkin"
		"name":"dodder_diff"
		"name":"dodder_edit"
		"name":"dodder_format_blob"
		"name":"dodder_new"
		"name":"dodder_query"
		"name":"dodder_read_checked_out"
		"name":"dodder_show"
		"name":"dodder_status"
		"name":"dodder_tag_query"
		"name":"dodder_type_query"
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
  write_mcp_tool_call_input "$input" dodder_show '{"object_id":"!md"}'

  run bash -o pipefail -c \
    'timeout 5s "'"$DODDER_BIN"'" mcp <"'"$input"'" | grep "\"id\":2" | grep -oE "\"text\":\"[^\"]*\""'

  assert_success
  refute_output '"text":""'
}

function mcp_query_empty_result_has_non_empty_text { # @test
  # Same root cause as the show-type test: an empty result set yields
  # empty stdout and an empty content block; assert the placeholder.
  # https://github.com/amarbel-llc/dodder/issues/213
  run_dodder_init_disable_age

  local input="$BATS_TEST_TMPDIR/mcp-query-empty.jsonrpc"
  write_mcp_tool_call_input "$input" dodder_query '{"query":[":z"]}'

  run bash -o pipefail -c \
    'timeout 5s "'"$DODDER_BIN"'" mcp <"'"$input"'" | grep "\"id\":2" | grep -oE "\"text\":\"[^\"]*\""'

  assert_success
  refute_output '"text":""'
}
