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

# Config mutation is log-only (FDR 0020): edit-config no longer writes a
# konfig object and is silent on success. The change is observed through
# show-config, the read surface.
function edit_config_and_change { # @test
	export EDITOR="bash -c 'echo \"# this is the body 2\" >> \"\$0\"'"
	run_dodder edit-config
	assert_success
	assert_output ''

	run_dodder show-config
	assert_success
	assert_line '# this is the body 2'
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
	assert_output ''

	# Verified through show-config (the config read surface). The legacy
	# `show :konfig` query is removed in a later task.
	run_dodder show-config
	assert_success
	assert_line 'print-time = false'
}

# After a real edit, the new config state is appended to the config log
# (FDR 0020), so `show-config` (no args) streams the new TOML blob and
# `show-config -history` lists the new entry as a box line. Config
# mutation is log-only — no konfig object is written.
function edit_config_show_config_roundtrips { # @test
	export EDITOR="bash -c 'echo \"# this is the body 2\" >> \"\$0\"'"
	run_dodder edit-config
	assert_success

	run_dodder show-config
	assert_success
	assert_output - <<-EOM
		blob-stores = [".default"]

		[defaults]
		type = "!md"
		tags = []

		[file-extensions]
		config = "konfig"
		conflict = "conflict"
		lockfile = "object-lockfile"
		organize = "md"
		repo = "repo"
		tag = "tag"
		type = "type"
		zettel = "zettel"

		[cli-output]
		print-blob_digests = true
		print-colors = true
		print-empty-blob_digests = false
		print-flush = true
		print-include-description = true
		print-include-types = true
		print-inventory_lists = true
		print-matched-dormant = false
		print-tags-always = true
		print-time = true
		print-unchanged = true

		[cli-output.abbreviations]
		zettel_ids = true
		merkle_ids = true

		[tools]
		merge = ["vimdiff"]
		# this is the body 2
	EOM

	# A single entry: this test runs against the committed fixture, which
	# predates init-seeding the config log, so the first real edit is the
	# only entry. (After the fixtures are regenerated with an init-seeded
	# root, this becomes two entries.) The blob digest is deterministic;
	# the tai timestamp and ed25519 signatures are not.
	run_dodder show-config -history
	assert_success
	assert_output --regexp '^\[konfig @blake2b256-wlqn0d2a583mpwq2h948eglrc26znyjuupzmsraqna6xszw99lfqeng70u [0-9.]+ dodder-repo-public_key-v1@ed25519_pub-[a-z0-9]+ dodder-object-sig-v2@ed25519_sig-[a-z0-9]+ !toml-config-v2\]$'
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

	# Config left the query surface (FDR 0020); the blob digest is recovered
	# from the config log via show-config -history rather than `:konfig`.
	run_dodder show-config -history
	assert_success
	digest="$(printf '%s\n' "$output" | head -n1 | grep -oE 'blake2b256-[a-z0-9]+' | head -n1)"
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
