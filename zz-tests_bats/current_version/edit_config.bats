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

# bats file_tags=user_story:config

function edit_config_and_change { # @test
	export EDITOR="bash -c 'echo \"# this is the body 2\" >> \"\$0\"'"
	run_dodder edit-config
	assert_success
	assert_output - <<-EOM
		[konfig @blake2b256-wlqn0d2a583mpwq2h948eglrc26znyjuupzmsraqna6xszw99lfqeng70u !toml-config-v2]
	EOM
}

function edit_config_and_dont_change { # @test
	export EDITOR="true"
	run_dodder edit-config
	assert_success
	assert_output ''
}

# The EDITOR temp file is the konfig blob verbatim — no `---` fences,
# no type directive — so the user is editing exactly what gets
# committed.
function edit_config_temp_file_is_bare_toml { # @test
	captured="$BATS_TEST_TMPDIR/captured.txt"
	export EDITOR="bash -c 'cp \"\$0\" \"$captured\"'"
	run_dodder edit-config
	assert_success
	assert_output ''

	run grep -c '^---$' "$captured"
	assert_failure
	assert_output '0'

	run grep -c '^! toml-config-v2$' "$captured"
	assert_failure
	assert_output '0'
}

# A value edited via `edit-config` must survive back through the load
# path. Toggles `print-time = true` to `print-time = false`, then
# reads the konfig back and confirms the new value is present.
function edit_config_value_roundtrips { # @test
	export EDITOR="bash -c 'sed -i \"s/print-time = true/print-time = false/\" \"\$0\"'"
	run_dodder edit-config
	assert_success
	assert_output --regexp '^\[konfig @blake2b256-[a-z0-9]+ !toml-config-v2\]$'

	run_dodder show -format text :konfig
	assert_success
	assert_line 'print-time = false'
}

# A freshly initialized repo writes the konfig blob as bare TOML — no
# hyphence wrapper inside the blob, no type directive — so the bytes
# `madder cat`s out match the bytes a user sees in `edit-config`.
function konfig_blob_fresh_init_is_bare_toml { # @test
	# setup() pre-populates BATS_TEST_TMPDIR with the fixture; init
	# must run from a fresh subdirectory to avoid the collision.
	mkdir -p "$BATS_TEST_TMPDIR/fresh"
	cd "$BATS_TEST_TMPDIR/fresh"
	run_dodder_init_disable_age test-bare-konfig

	run_dodder show -format object-id-blob-digest :konfig
	assert_success
	digest="${output##* }"
	[[ -n $digest ]] || fail "could not parse digest from: $output"

	blob_path="$BATS_TEST_TMPDIR/konfig-blob.bin"
	run_madder cat .default "$digest"
	assert_success
	printf '%s\n' "$output" >"$blob_path"

	run grep -c '^---$' "$blob_path"
	assert_failure
	assert_output '0'

	run grep -c '^! toml-config-v2$' "$blob_path"
	assert_failure
	assert_output '0'

	run grep -m1 -E '^[a-z]' "$blob_path"
	assert_success
}
