#! /usr/bin/env bats

setup() {
	load "$(dirname "$BATS_TEST_FILE")/common.bash"

	# for shellcheck SC2154
	export output

	copy_from_version "$DIR"
}

teardown() {
	chflags_and_rm
}

function complete_show { # @test
	skip                    # TODO add back support
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
	skip                            # TODO add back support
	run_dodder complete show :z
	assert_success
	assert_output_unsorted --regexp - <<-EOM
		one/dos.*Zettel: !md wow ok again
		one/uno.*Zettel: !md wow the first
	EOM
}

function complete_show_types { # @test
	skip                          # TODO add back support
	run_dodder complete show :t
	assert_success
	assert_output_unsorted --regexp - <<-EOM
		md.*Type
	EOM
}

function complete_show_tags { # @test
	skip                         # TODO add back support
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
	assert_output_unsorted --regexp - <<-'EOM'
		add
		blob_store-cat
		blob_store-cat-ids
		blob_store-complete.*complete a command-line
		blob_store-fsck
		blob_store-info-repo
		blob_store-init
		blob_store-init-from
		blob_store-init-pointer
		blob_store-init-sftp-explicit
		blob_store-init-sftp-ssh_config
		blob_store-list
		blob_store-pack
		blob_store-read
		blob_store-sync
		blob_store-write
		cat-alfred
		checkin
		checkin-blob
		checkin-json
		checkout
		clean
		clone
		complete.*complete a command-line
		debug-print-probe-index
		deinit
		diff
		dormant-add
		dormant-edit
		dormant-remove
		edit
		edit-config
		exec
		export
		find-missing
		format-blob
		format-object
		format-organize
		fsck
		gen
		import
		info
		info-repo
		info-workspace
		init
		init-workspace
		last
		merge-tool
		new
		organize
		peek-zettel-ids
		pull
		pull-blob-store
		push
		reindex
		remote-add
		repo-fsck
		revert
		save
		serve
		show
		status
		update
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
