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
		"name":"import"
		"name":"new"
		"name":"organize_commit"
		"name":"organize_plan"
		"name":"pull"
		"name":"push"
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
		"name":"import"
		"name":"new"
		"name":"organize_commit"
		"name":"organize_plan"
		"name":"pull"
		"name":"push"
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

# #7: organize_plan renders matching objects as an organize buffer. This
# covers the MCP wiring (tool reachable, returns non-empty text content); the
# organize round-trip behavior itself is exercised exhaustively by the CLI
# `organize -mode commit-directly` tests in organize.bats, which share the
# repo_actions.OrganizePlan / OrganizeCommitFromReader path this tool uses.
function mcp_organize_plan_returns_buffer { # @test
  run_dodder_init_disable_age

  run_dodder new -edit=false - <<-EOM
		---
		# organize me
		- todo
		! md
		---

		body
	EOM
  assert_success

  local input="$BATS_TEST_TMPDIR/mcp-organize-plan.jsonrpc"
  write_mcp_tool_call_input "$input" organize_plan '{"query":[":z"]}'

  run bash -o pipefail -c \
    'timeout 15s "'"$DODDER_BIN"'" mcp <"'"$input"'" | grep "\"id\":2" | grep -oE "\"text\":\"[^\"]*\""'

  assert_success
  # The buffer is non-empty and mentions the zettel's description.
  refute_output '"text":""'
  assert_output --regexp 'organize me'
}

# #7: organize_commit applies an edited organize buffer that adds a tag
# heading across the matched objects. The buffer is a single-line heading
# followed by the object lines, kept simple enough to embed inline in the
# JSON-RPC argument. Asserts the commit reports success and the tag lands —
# proving the `organize` argument flows through the MCP layer into the parser
# and commit. (The full breadth of organize edits — moves, descriptions,
# merges — is covered by the CLI organize.bats over the shared commit path.)
function mcp_organize_commit_applies_tag { # @test
  run_dodder_init_disable_age

  run_dodder new -edit=false - <<-EOM
		---
		# commit me
		- todo
		! md
		---

		body
	EOM
  assert_success

  # Render the canonical organize buffer (same content organize_plan returns)
  # and write the edited form — a new heading tag over the existing lines — to
  # a file, then JSON-encode it for the tool argument using the dodder binary's
  # own JSON checkin encoder is overkill; use printf + sed for the minimal
  # escaping the buffer needs (no embedded quotes/backslashes in fixture text).
  run_dodder organize -mode output-only :z
  assert_success

  local edited="$BATS_TEST_TMPDIR/edited-organize"
  {
    echo "# mcp-applied"
    printf '%s\n' "$output"
  } >"$edited"

  # Minimal JSON string encoding: escape backslashes, quotes, then turn
  # newlines into \n. The fixture buffer contains none of the former, so this
  # is exact for this test's content.
  local buffer_json
  buffer_json="\"$(sed -e 's/\\/\\\\/g' -e 's/"/\\"/g' "$edited" | awk 'BEGIN{ORS="\\n"} {print}')\""

  local commit_input="$BATS_TEST_TMPDIR/mcp-organize-commit.jsonrpc"
  write_mcp_tool_call_input "$commit_input" organize_commit \
    '{"query":[":z"],"organize":'"$buffer_json"'}'

  run bash -o pipefail -c \
    'timeout 15s "'"$DODDER_BIN"'" mcp <"'"$commit_input"'" | grep "\"id\":2"'
  assert_success
  assert_output --regexp 'organize committed'

  # The tag landed on the zettel.
  run_dodder show -format text :z
  assert_success
  assert_output --partial 'mcp-applied'
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
  # unwired scope (remote-first /name) is rejected by repo_id.CheckSupported.
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

  # remote-first /name is parsed but not yet resolvable (no remote
  # transport) -> CheckSupported reject. (Forced-system //name now
  # resolves, #280, so it's no longer the rejection probe here.)
  local in_remote="$BATS_TEST_TMPDIR/mcp-q-remote.jsonrpc"
  write_mcp_tool_call_input "$in_remote" query '{"query":[":z"],"repo_id":"/backup"}'
  run bash -o pipefail -c \
    'timeout 5s "'"$DODDER_BIN"'" mcp <"'"$in_remote"'" | grep "\"id\":2"'
  assert_success
  assert_output --regexp 'remote-first.*not yet resolvable'
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
  # the server's cwd repo is listed with its routable `.default` spelling
  # (FDR-0019 #276), so the listing's URI round-trips back to the repo
  assert_output --regexp 'repos/\.default'

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

# bats test_tags=repo_id
function mcp_repos_lists_both_scopes { # @test
  # FDR-0019 #276: dodder:///repos lists both scopes — cwd repos spelled
  # .name, XDG-user repos spelled name — like `info-repo repos`. With a
  # user-scope repo and the server's own cwd repo both present, both appear
  # (the server is bound to the cwd scope but enumerates the user scope via
  # a no-walk-up env_dir).
  run_dodder init \
    -yin <(cat_yin) \
    -yang <(cat_yang) \
    -encryption none \
    -repo_id userrepo \
    test-repo-id
  assert_success

  run_dodder_init_disable_age

  local in_repos="$BATS_TEST_TMPDIR/mcp-both-repos.jsonrpc"
  write_mcp_resource_read_input "$in_repos" 'dodder:///repos'
  run bash -o pipefail -c \
    'timeout 5s "'"$DODDER_BIN"'" mcp <"'"$in_repos"'" | grep "\"id\":2"'
  assert_success
  assert_output --regexp 'total_repos[^0-9]*[2-9]'
  assert_output --regexp 'repos/\.default'
  assert_output --regexp 'repos/userrepo'
}

# bats test_tags=repo_id
function mcp_bare_name_routes_to_user_scope_not_workspace { # @test
  # Regression: from inside a .dodder/ workspace, the bridge must route a
  # bare-name repo_id (`default`, XDG-user scope) to the USER repo, while the
  # cwd spelling (`.default`) stays on the workspace repo. Pre-fix both
  # collapsed to the workspace repo because MakeDefault's cwd walk-up hijacked
  # the explicit bare name (FDR-0019: bare name is XDG-user scope
  # unconditionally).

  # user-scope `default` with a distinctive marker, before any cwd workspace.
  run_dodder init -yin <(cat_yin) -yang <(cat_yang) -encryption none \
    -repo_id default user-default-id
  assert_success
  user_zettel="$(mktemp)"
  {
    echo "---"
    echo "# user scope marker"
    echo "- task"
    echo "! md"
    echo "---"
  } >"$user_zettel"
  run_dodder new -edit=false -repo_id default "$user_zettel"
  assert_success

  # the server's own cwd repo (.default) with a different marker.
  run_dodder_init_disable_age
  cwd_zettel="$(mktemp)"
  {
    echo "---"
    echo "# cwd scope marker"
    echo "- task"
    echo "! md"
    echo "---"
  } >"$cwd_zettel"
  run_dodder new -edit=false "$cwd_zettel"
  assert_success

  # repo_id=default -> the USER repo -> user marker, not cwd marker.
  local in_user="$BATS_TEST_TMPDIR/mcp-q-user.jsonrpc"
  write_mcp_tool_call_input "$in_user" query '{"query":[":z"],"repo_id":"default"}'
  run bash -o pipefail -c \
    'timeout 5s "'"$DODDER_BIN"'" mcp <"'"$in_user"'" | grep "\"id\":2"'
  assert_success
  assert_output --regexp 'user scope marker'
  refute_output --regexp 'cwd scope marker'

  # repo_id=.default -> the workspace repo -> cwd marker, not user marker.
  local in_cwd="$BATS_TEST_TMPDIR/mcp-q-cwd.jsonrpc"
  write_mcp_tool_call_input "$in_cwd" query '{"query":[":z"],"repo_id":".default"}'
  run bash -o pipefail -c \
    'timeout 5s "'"$DODDER_BIN"'" mcp <"'"$in_cwd"'" | grep "\"id\":2"'
  assert_success
  assert_output --regexp 'cwd scope marker'
  refute_output --regexp 'user scope marker'
}

# bats test_tags=repo_id
function mcp_reset_lock_routes_by_repo_id { # @test
  # FDR-0019 #278: reset-lock gains an optional repo_id and opens that repo
  # per call. Targeting the server's own cwd repo (.default) clears its
  # staged lock; a nonexistent repo errors at open (proving the param
  # selects the repo, not just decorates the call).
  run_dodder_init_disable_age

  run_dodder info xdg
  assert_success
  local state_home
  state_home="$(echo "$output" | grep -E 'STATE' | head -n1 | cut -d= -f2 | tr -d '"')"
  assert [ -n "$state_home" ]
  mkdir -p "$state_home"
  touch "$state_home/lock"

  local in_default="$BATS_TEST_TMPDIR/mcp-rl-default.jsonrpc"
  write_mcp_tool_call_input "$in_default" reset-lock '{"repo_id":".default"}'
  run bash -o pipefail -c \
    'timeout 15s "'"$DODDER_BIN"'" mcp <"'"$in_default"'" | grep "\"id\":2"'
  assert_success
  assert_output --regexp 'no longer locked'
  assert [ ! -e "$state_home/lock" ]

  local in_missing="$BATS_TEST_TMPDIR/mcp-rl-missing.jsonrpc"
  write_mcp_tool_call_input "$in_missing" reset-lock '{"repo_id":"nonexistent"}'
  run bash -o pipefail -c \
    'timeout 15s "'"$DODDER_BIN"'" mcp <"'"$in_missing"'" | grep "\"id\":2"'
  assert_success
  assert_output --regexp 'not in a dodder directory|repos/nonexistent'
}

# bats test_tags=repo_id
function mcp_edit_and_blob_formats_route_by_repo_id { # @test
  # FDR-0019 #278: edit and blob-format listing open the addressed repo per
  # call. edit with repo_id=.default mutates the server's repo; a
  # nonexistent repo errors at open. Reading a scoped repo's blob formats no
  # longer returns the old "per-repo not yet supported" deferral.
  run_dodder_init_disable_age

  to_add="$(mktemp)"
  {
    echo "---"
    echo "# repo 278 probe"
    echo "- task"
    echo "! md"
    echo "---"
  } >"$to_add"
  run_dodder new -edit=false "$to_add"
  assert_success

  run_dodder show -format json :z
  assert_success
  local object_id
  object_id="$(echo "$output" | grep -oE '"object-id":"[^"]*"' | head -1 | sed 's/.*:"//;s/"$//')"
  assert [ -n "$object_id" ]

  # edit via repo_id=.default updates the description
  local in_edit="$BATS_TEST_TMPDIR/mcp-edit.jsonrpc"
  write_mcp_tool_call_input "$in_edit" edit \
    '{"object_id":"'"$object_id"'","description":"edited via 278","repo_id":".default"}'
  run bash -o pipefail -c \
    'timeout 15s "'"$DODDER_BIN"'" mcp <"'"$in_edit"'" | grep "\"id\":2"'
  assert_success
  assert_output --regexp 'edited via 278'

  # edit targeting a nonexistent repo errors at open
  local in_edit_missing="$BATS_TEST_TMPDIR/mcp-edit-missing.jsonrpc"
  write_mcp_tool_call_input "$in_edit_missing" edit \
    '{"object_id":"'"$object_id"'","description":"x","repo_id":"nonexistent"}'
  run bash -o pipefail -c \
    'timeout 15s "'"$DODDER_BIN"'" mcp <"'"$in_edit_missing"'" | grep "\"id\":2"'
  assert_success
  assert_output --regexp 'not in a dodder directory|repos/nonexistent'

  # blob-format listing for the scoped repo no longer returns the deferral
  local in_fmt="$BATS_TEST_TMPDIR/mcp-fmt.jsonrpc"
  write_mcp_resource_read_input "$in_fmt" \
    "dodder:///repos/.default/objects/$object_id/blob/formats"
  run bash -o pipefail -c \
    'timeout 5s "'"$DODDER_BIN"'" mcp <"'"$in_fmt"'" | grep "\"id\":2"'
  assert_success
  refute_output --regexp 'per-repo not yet supported'
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

# The new tool's object_id + blob args author a type object (e.g. !task)
# directly: the chosen id is honored, the meta-type is set from the genre
# (!toml-type-v2), and the blob body is written. Mirrors the CLI
# new_object_id_type_with_blob test through the MCP surface.
function mcp_new_object_id_authors_type_with_blob { # @test
  run_dodder_init_disable_age

  local input="$BATS_TEST_TMPDIR/mcp-new-task-type.jsonrpc"
  write_mcp_tool_call_input "$input" new \
    '{"object_id":"!task","blob":"file-extension = \"toml\"\n"}'

  run bash -o pipefail -c \
    'timeout 5s "'"$DODDER_BIN"'" mcp <"'"$input"'" | grep "\"id\":2"'
  assert_success
  assert_output --regexp '\[!task @blake2b256-[a-z0-9]+ !toml-type-v2\]'

  # the type is queryable afterward, proving it committed
  run_dodder show '!task:t'
  assert_success
  assert_output --regexp '^\[!task @blake2b256-.+ !toml-type-v2\]$'
}

# The push tool transfers objects to a local repo via -direct, exercising
# the remote-transfer MCP wiring and the direct/query arg translation.
# Mirrors the CLI push_direct_local_path_no_conflicts test through the MCP
# surface. push works without a workspace, so the fixture repo (no
# .dodder-workspace) is a valid source.
function mcp_push_direct_transfers_to_local_repo { # @test
  copy_from_version "$DIR"

  (
    mkdir -p them
    pushd them || exit 1
    run_dodder_init
    popd || exit 1
  )

  local input="$BATS_TEST_TMPDIR/mcp-push.jsonrpc"
  write_mcp_tool_call_input "$input" push \
    '{"direct":"'"$(realpath them)"'","query":["+zettel,typ,etikett"]}'

  run timeout 10s "$DODDER_BIN" mcp <"$input"
  assert_success

  pushd them || exit 1
  run_dodder show +zettel,typ,etikett
  assert_success
  assert_output_unsorted - <<-EOM
		[!md @$(get_type_blob_sha) !toml-type-v2]
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
		[one/uno @blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0 !md "wow ok" tag-1 tag-2]
	EOM
  popd || exit 1
}

# The import tool ingests an inventory list into the local store via the
# MCP surface, exercising importToolCLIArgs (paths + blob_store_id). The
# inner repo's MCP server is pinned to inner via the ceiling vars so its
# walk-up does not discover the outer fixture repo. Mirrors the CLI import
# test.
function mcp_import_from_inventory_list { # @test
  run_madder init shared
  assert_success

  run_dodder init \
    -yin <(cat_yin) \
    -yang <(cat_yang) \
    -repo_id .default \
    -encryption none \
    -blob_store-id shared \
    test
  assert_success

  run_dodder init-workspace -experimental-repo=false
  create_test_zettels

  run_dodder export -print-time=true +z,e,t
  assert_success
  echo "$output" >list
  list="$(realpath list)"

  (
    mkdir inner
    pushd inner || exit 1
    run_dodder_init
    popd || exit 1
  )

  local input="$BATS_TEST_TMPDIR/mcp-import.jsonrpc"
  write_mcp_tool_call_input "$input" import \
    '{"paths":["'"$list"'"],"blob_store_id":"shared"}'

  pushd inner || exit 1
  run env \
    DODDER_CEILING_DIRECTORIES="$PWD" \
    MADDER_CEILING_DIRECTORIES="$PWD" \
    timeout 10s "$DODDER_BIN" mcp <"$input"
  assert_success

  run_dodder show one/uno
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM
  popd || exit 1
}

# The import tool's dry_run param previews the plan without committing. The
# plan must come back in the tool result (proving it is routed to the
# captured UI writer, not os.Stderr), and nothing is committed.
function mcp_import_dry_run { # @test
  run_madder init shared
  assert_success

  run_dodder init \
    -yin <(cat_yin) \
    -yang <(cat_yang) \
    -repo_id .default \
    -encryption none \
    -blob_store-id shared \
    test
  assert_success

  run_dodder init-workspace -experimental-repo=false
  create_test_zettels

  run_dodder export -print-time=true +z,e,t
  assert_success
  echo "$output" >list
  list="$(realpath list)"

  (
    mkdir inner
    pushd inner || exit 1
    run_dodder_init
    popd || exit 1
  )

  local input="$BATS_TEST_TMPDIR/mcp-import-dry-run.jsonrpc"
  write_mcp_tool_call_input "$input" import \
    '{"paths":["'"$list"'"],"blob_store_id":"shared","dry_run":true}'

  pushd inner || exit 1
  run env \
    DODDER_CEILING_DIRECTORIES="$PWD" \
    MADDER_CEILING_DIRECTORIES="$PWD" \
    timeout 10s "$DODDER_BIN" mcp <"$input"
  assert_success
  # The plan summary (FormatSummary) is in the result text.
  assert_output --partial "committable"

  # Nothing committed: inner still has only its init type.
  run_dodder show :z,e,t
  assert_success
  assert_output - <<-EOM
		[!md @$(get_type_blob_sha) !toml-type-v2]
	EOM
  popd || exit 1
}

# The import tool's blobless_type_remapping param resolves a blobless type by
# remapping it onto a local type (proving the translator emits
# -resolve-blobless-type). Mirrors the CLI import_resolve_blobless_type test.
function mcp_import_resolve_blobless_type { # @test
  run_madder init shared
  assert_success

  run_dodder init \
    -yin <(cat_yin) \
    -yang <(cat_yang) \
    -repo_id .default \
    -encryption none \
    -blob_store-id shared \
    test
  assert_success

  run_dodder init-workspace -experimental-repo=false
  create_test_zettels

  run_dodder new -edit=false -object-id '!custom'
  assert_success

  {
    echo "---"
    echo "# custom typed zettel"
    echo "! custom"
    echo "---"
    echo
    echo "custom body"
  } >to_add

  run_dodder new -edit=false to_add
  assert_success

  run_dodder export -print-time=true +z,e,t
  assert_success
  echo "$output" >list
  list="$(realpath list)"

  (
    mkdir inner
    pushd inner || exit 1
    run_dodder_init
    popd || exit 1
  )

  local input="$BATS_TEST_TMPDIR/mcp-import-blobless.jsonrpc"
  write_mcp_tool_call_input "$input" import \
    '{"paths":["'"$list"'"],"blob_store_id":"shared","blobless_type_remapping":{"!custom":"!md"}}'

  pushd inner || exit 1
  run env \
    DODDER_CEILING_DIRECTORIES="$PWD" \
    MADDER_CEILING_DIRECTORIES="$PWD" \
    timeout 10s "$DODDER_BIN" mcp <"$input"
  assert_success

  # The blobless !custom type was remapped to !md, so no object remains
  # typed !custom.
  run_dodder show :z
  assert_success
  refute_output --partial '!custom'
  popd || exit 1
}
