#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

  # for shellcheck SC2154
  export output

  copy_from_version "$DIR"
}

teardown() {
  chflags_nouchg
}

function complete_show { # @test
  skip                   # TODO add back support
  run_dodder complete show --
  assert_success
  assert_output_unsorted --regexp - <<-'EOM'
		tag-1.*Tag
		tag-2.*Tag
		tag-3.*Tag
		tag-4.*Tag
		tag.*Tag
	EOM
}

function complete_show_all { # @test
  skip
  run_dodder complete show :z,t,b,e
  assert_success
  assert_output_unsorted --regexp - <<-EOM
		-after
		-before
		-exclude-recognized
		-exclude-untracked
		-format.*format
		-kasten.*none or Browser
		.*InventoryList
		.*InventoryList
		.*InventoryList
		.*InventoryList
		!md.*Type
		one/dos.*Zettel: !md wow ok again
		one/uno.*Zettel: !md wow the first
		tag.*Tag
		tag.1.*Tag
		tag.2.*Tag
		tag.3.*Tag
		tag.4.*Tag
	EOM
}

function complete_show_zettels { # @test
  skip                           # TODO add back support
  run_dodder complete show :z
  assert_success
  assert_output_unsorted --regexp - <<-EOM
		one/dos.*Zettel: !md wow ok again
		one/uno.*Zettel: !md wow the first
	EOM
}

function complete_show_types { # @test
  skip                         # TODO add back support
  run_dodder complete show :t
  assert_success
  assert_output_unsorted --regexp - <<-EOM
		md.*Type
	EOM
}

function complete_show_tags { # @test
  skip                        # TODO add back support
  run_dodder complete show :e
  assert_success
  assert_output_unsorted --regexp - <<-EOM
		tag-3.*Tag
		tag-4.*Tag
	EOM
}

function complete_subcmd { # @test
  run_dodder complete
  assert_success
  # Use per-line `assert_line --regexp` instead of
  # `assert_output_unsorted --regexp` so the assertion is locale-
  # collation independent. The latter sorts both the regex and the
  # actual output before matching, and `.` (regex meta) sorts
  # differently than the actual whitespace padding under the C locale,
  # producing a sort-order mismatch even when every command is present.
  while IFS= read -r line; do
    [[ -z $line ]] && continue
    assert_line --regexp "$line"
  done <<-'EOM'
		^add[[:space:]]+commit workspace changes to the store$
		^add-zettel-ids-yang[[:space:]]+add yang words to the zettel id pool$
		^add-zettel-ids-yin[[:space:]]+add yin words to the zettel id pool$
		^cat-alfred[[:space:]]+output objects in Alfred workflow format$
		^cat-object[[:space:]]+output raw object content by markl id$
		^check-workspace[[:space:]]+check workspace state$
		^checkin[[:space:]]+commit workspace changes to the store$
		^checkin-blob[[:space:]]+commit blob changes with metadata updates$
		^checkin-json[[:space:]]+commit objects from JSON on stdin$
		^checkout[[:space:]]+check out objects to the workspace$
		^clean[[:space:]]+remove checked-out objects from the workspace$
		^clone[[:space:]]+clone a remote repository$
		^complete[[:space:]]+complete a command-line$
		^debug-print-probe-index[[:space:]]+print stream index probes$
		^deinit[[:space:]]+remove repository and workspace directories$
		^diff[[:space:]]+show differences between workspace and store$
		^dormant-add[[:space:]]+add tags to the dormant index$
		^dormant-edit[[:space:]]+edit dormant tags in an editor$
		^dormant-remove[[:space:]]+remove tags from the dormant index$
		^edit[[:space:]]+check out and edit objects in an editor$
		^edit-config[[:space:]]+edit the repository configuration$
		^exec[[:space:]]+execute a script stored as a blob$
		^export[[:space:]]+export objects to an inventory list archive$
		^find-missing[[:space:]]+find blob digests missing from stores$
		^format-blob[[:space:]]+format an object's blob content$
		^format-object[[:space:]]+format an object with a type formatter$
		^format-organize[[:space:]]+format an organize file$
		^fsck[[:space:]]+verify object integrity across stores$
		^gen[[:space:]]+generate cryptographic keys$
		^generate-zettel-id-components[[:space:]]+extract unique zettel id components from stdin$
		^hook[[:space:]]+respond to a Claude Code hook event from the clown plugin$
		^import[[:space:]]+import objects from inventory list files$
		^info[[:space:]]+display repository information$
		^info-pivy_agent[[:space:]]+list ECDSA keys in pivy-agent$
		^info-ssh_agent[[:space:]]+list keys in the SSH agent$
		^info-repo[[:space:]]+display repository configuration$
		^info-workspace[[:space:]]+display workspace configuration$
		^init[[:space:]]+initialize a new repository$
		^init-default[[:space:]]+initialize a repository with sensible defaults$
		^init-from[[:space:]]+create a new repo copy-migrated from an existing local repo \(fresh instance identity, same keys; source untouched\)$
		^init-workspace[[:space:]]+initialize a workspace directory$
		^init-workspace-default[[:space:]]+initialize a workspace with default ignore patterns$
		^install-mcp[[:space:]]+install MCP server configuration$
		^last[[:space:]]+display the most recently committed objects$
		^mcp[[:space:]]+start the MCP server$
		^merge-tool[[:space:]]+resolve merge conflicts with an external tool$
		^migrate-config-seed-key[[:space:]]+re-encode config-seed private key in canonical split-HRP form$
		^migrate-repo-layout[[:space:]]+copy a legacy flat .dodder tree into the repos/<name>/ nested layout \(FDR-0019\); never modifies -source$
		^migrate-zettel-ids[[:space:]]+migrate zettel id flat files to log format$
		^new[[:space:]]+create new zettels$
		^organize[[:space:]]+organize objects with a text editor$
		^peek-zettel-ids[[:space:]]+preview available zettel ids$
		^pull[[:space:]]+pull objects from a remote repository$
		^pull-blob-store[[:space:]]+pull blobs from a remote blob store$
		^push[[:space:]]+push objects to a remote repository$
		^reconcile-blob-to-store[[:space:]]+TEMPORARY workaround \(dodder#359\): copy one blob by digest into an explicitly-addressed destination store$
		^reindex[[:space:]]+rebuild store indices$
		^remote-add[[:space:]]+add a remote repository$
		^repair-legacy-config-blob[[:space:]]+ONE-TIME: repair a header-corrupted legacy \(pre-config_log\) config blob and bootstrap config_log$
		^repo-fsck[[:space:]]+verify repository inventory list integrity$
		^revert[[:space:]]+revert objects to their stored state$
		^save[[:space:]]+commit workspace changes to the store$
		^serve[[:space:]]+start the HTTP server$
		^serve-proto[[:space:]]+serve the drtp remote transfer protocol \(websocket-capable\)$
		^set-parent[[:space:]]+pin the workspace's parent repo identity \(pubkey\)$
		^show[[:space:]]+display objects from the store$
		^show-config[[:space:]]+read the repository configuration log$
		^status[[:space:]]+show workspace object state$
		^transform[[:space:]]+run a Lua list-in/list-out transform over queried objects and commit the result$
		^update[[:space:]]+update type lock signatures$
		^version[[:space:]]+print dodder build version and commit$
	EOM
}

function complete_complete { # @test
  run_dodder complete complete
  assert_success
  assert_output_unsorted --regexp - <<-'EOM'
		-bash-style.*
		-in-progress.*
	EOM
}

function complete_init_workspace { # @test
  run_dodder complete init-workspace
  assert_success

  # shellcheck disable=SC2016
  assert_output --regexp -- '-query.*default query for `show`'
  # shellcheck disable=SC2016
  assert_output --regexp -- '-tags.*tags added for new objects in `checkin`, `new`, `organize`'
  # shellcheck disable=SC2016
  assert_output --regexp -- '-type.*type used for new objects in `new` and `organize`'

  skip # TODO add back support
  run_dodder complete init-workspace -tags
  assert_success
  assert_output_unsorted --regexp - <<-'EOM'
		tag-1.*Tag
		tag-2.*Tag
		tag-3.*Tag
		tag-4.*Tag
		tag.*Tag
	EOM

  run_dodder complete init-workspace -query
  assert_success
  assert_output_unsorted --regexp - <<-'EOM'
		tag-1.*Tag
		tag-2.*Tag
		tag-3.*Tag
		tag-4.*Tag
		tag.*Tag
	EOM

  run_dodder complete init-workspace -type
  assert_success
  assert_output_unsorted --regexp - <<-'EOM'
		!md.*Type
	EOM

  run_dodder complete -in-progress="tag" init-workspace -tags tag
  assert_success
  assert_output_unsorted --regexp - <<-'EOM'
		tag-1.*Tag
		tag-2.*Tag
		tag-3.*Tag
		tag-4.*Tag
		tag.*Tag
	EOM

  mkdir -p workspaces/test

  run_dodder complete -in-progress="workspaces" init-workspace -tags tag workspaces
  assert_success

  # shellcheck disable=SC2016
  assert_output_unsorted --regexp -- '-query.*default query for `show`'
  # shellcheck disable=SC2016
  assert_output_unsorted --regexp -- '-tags.*tags added for new objects in `checkin`, `new`, `organize`'
  # shellcheck disable=SC2016
  assert_output_unsorted --regexp -- 'test/.*directory'
  # shellcheck disable=SC2016
  assert_output_unsorted --regexp -- '-type.*type used for new objects in `new` and `organize`'
}

function complete_repo_fsck { # @test
  run_dodder complete repo-fsck
  assert_success
  assert_output --regexp '\.default'
}

function complete_repo_id { # @test
  run_dodder complete show -repo_id
  assert_success
  assert_output --regexp '\.default'
}

function complete_checkin { # @test
  touch wow.md
  run_dodder complete checkin -organize -delete
  assert_success

  # shellcheck disable=SC2016
  assert_output --regexp -- 'wow.md.*file'

  touch wow.md
  run_dodder complete checkin -organize -delete --
  assert_success

  # shellcheck disable=SC2016
  assert_output --regexp -- 'wow.md.*file'
}
