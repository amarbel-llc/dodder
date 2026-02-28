#! /usr/bin/env bats

# bats file_tags=ecdsa_p256,af_unix

setup() {
	load "$(dirname "$BATS_TEST_FILE")/common.bash"

	# for shellcheck SC2154
	export output

	# Unix domain sockets have a 108-char path limit, so use /tmp for the
	# agent socket (BATS_TEST_TMPDIR paths are too long).
	_agent_dir="$(mktemp -d /tmp/bats-ssh-XXXXXX)"

	ssh-agent -a "$_agent_dir/agent.sock" -s >"$BATS_TEST_TMPDIR/agent-env" 2>&1

	eval "$(cat "$BATS_TEST_TMPDIR/agent-env")"

	# Generate ECDSA P-256 key (no passphrase)
	ssh-keygen -t ecdsa -b 256 -f "$BATS_TEST_TMPDIR/id_ecdsa" -N "" -q

	# Load into test agent
	ssh-add "$BATS_TEST_TMPDIR/id_ecdsa"
}

teardown() {
	# Kill test agent and clean up socket dir
	if [[ -n "${SSH_AGENT_PID:-}" ]]; then
		ssh-agent -k >/dev/null 2>&1 || true
	fi

	rm -rf "${_agent_dir:-}" 2>/dev/null || true

	chflags_and_rm
}

function init_with_ecdsa_p256_ssh_key { # @test
	# Discover the key's markl ID via dodder
	run_dodder info-ssh_agent
	assert_success
	assert_output --regexp '^ecdsa_p256_ssh-.+'

	ecdsa_key="$output"

	# Init repo with ECDSA P256 key
	run_dodder init \
		-yin <(cat_yin) \
		-yang <(cat_yang) \
		-override-xdg-with-cwd \
		-lock-internal-files=false \
		-private_key "$ecdsa_key" \
		test-ecdsa

	assert_success

	# Verify objects can be read back (signature verification passes)
	run_dodder show -format log :konfig
	assert_success
	assert_output --regexp '\[konfig @.+ !toml-config-v2\]'

	# `last` decodes the most recent inventory list entry, which
	# includes full signature verification
	run_dodder last
	assert_success
	assert_output

	# `fsck` validates all objects in the store
	run_dodder fsck
	assert_success
	assert_output --partial "TAP version 14"
	refute_output --partial "not ok"
}
