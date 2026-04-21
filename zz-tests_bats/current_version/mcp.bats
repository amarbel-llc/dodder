#! /usr/bin/env bats

setup() {
	load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

	# for shellcheck SC2154
	export output
}

teardown() {
	chflags_nouchg
}

# bats file_tags=user_story:mcp

# Extract tool names from a captured dodder MCP tools/list response, one per
# line, sorted and deduplicated. Parses raw JSON with grep rather than jq so
# the test has no jq dependency.
function extract_tool_names {
	grep -oE '"name":"dodder_[a-z_]+"' | sort -u
}

export -f extract_tool_names

function mcp_initialize_no_workspace { # @test
	# Outside a workspace, the MCP server must still start and answer the
	# initialize call cleanly — otherwise every moxy/Claude Code session
	# launched from a non-workspace directory loses dodder entirely.
	# https://github.com/amarbel-llc/dodder/issues/116
	copy_from_version "$DIR"

	run bash -c 'printf "%s\n" \
		'\''{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"0.1.0"}}}'\'' \
		| timeout 5s "$DODDER_BIN" mcp'

	assert_success
	assert_output --regexp '"serverInfo":\{[^}]*"name":"dodder"'
	assert_output --regexp '"protocolVersion":"2024-11-05"'
}

function mcp_tools_list_no_workspace { # @test
	# Outside a workspace, workspace-scoped tools (status, checkin, diff,
	# read_checked_out) must not appear in tools/list.
	# https://github.com/amarbel-llc/dodder/issues/116
	copy_from_version "$DIR"

	run bash -c 'printf "%s\n" \
		'\''{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"0.1.0"}}}'\'' \
		'\''{"jsonrpc":"2.0","method":"notifications/initialized"}'\'' \
		'\''{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}'\'' \
		| timeout 5s "$DODDER_BIN" mcp \
		| extract_tool_names'

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
	# Inside a workspace, the full tool set is advertised.
	# https://github.com/amarbel-llc/dodder/issues/116
	run_dodder_init_disable_age

	run bash -c 'printf "%s\n" \
		'\''{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"0.1.0"}}}'\'' \
		'\''{"jsonrpc":"2.0","method":"notifications/initialized"}'\'' \
		'\''{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}'\'' \
		| timeout 5s "$DODDER_BIN" mcp \
		| extract_tool_names'

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
