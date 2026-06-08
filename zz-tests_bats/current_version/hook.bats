#! /usr/bin/env bats

# Pins the CLI seam of the clown plugin's PreToolUse hook (#244): the
# plugin's hooks/handler script execs `dodder hook` with the Claude
# Code hook event on stdin. The decision table itself is unit-tested
# in go/internal/bravo/claude_hooks; these tests cover the stdin ->
# subcommand -> stdout wiring through the real binary.

setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

  # for shellcheck SC2154
  export output
}

function hook_allows_read_only_mcp_tool { # @test
  run_dodder hook <<-EOM
		{"hook_event_name": "PreToolUse", "tool_name": "mcp__plugin_dodder_dodder__show", "tool_input": {}}
	EOM
  assert_success
  assert_output '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"allow","permissionDecisionReason":"read-only dodder MCP tool, cannot mutate the store"}}'
}

function hook_falls_through_for_mutating_mcp_tool { # @test
  run_dodder hook <<-EOM
		{"hook_event_name": "PreToolUse", "tool_name": "mcp__plugin_dodder_dodder__checkin", "tool_input": {}}
	EOM
  assert_success
  assert_output ''
}

function hook_falls_through_for_unrelated_tool { # @test
  run_dodder hook <<-EOM
		{"hook_event_name": "PreToolUse", "tool_name": "Bash", "tool_input": {"command": "ls"}}
	EOM
  assert_success
  assert_output ''
}

function hook_asks_for_reset_lock { # @test
  # reset-lock forcibly breaks the repo's env lock; the hook must force
  # a user prompt on every call (#249).
  run_dodder hook <<-EOM
		{"hook_event_name": "PreToolUse", "tool_name": "mcp__plugin_dodder_dodder__reset-lock", "tool_input": {}}
	EOM
  assert_success
  assert_output '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"ask","permissionDecisionReason":"reset-lock forcibly breaks the repo'"'"'s environment lock; every invocation requires explicit user approval"}}'
}
