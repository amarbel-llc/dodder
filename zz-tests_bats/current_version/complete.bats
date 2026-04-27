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
  assert_output_unsorted --regexp - <<-'EOM'
		add.*commit workspace changes to the store
		add-zettel-ids-yang.*add yang words to the zettel id pool
		add-zettel-ids-yin.*add yin words to the zettel id pool
		blob_store-cat.*output blob contents by digest
		blob_store-cat-ids.*list all blob digests in a store
		blob_store-encode-ids.*convert hex digests to native markl IDs
		blob_store-complete.*complete a command-line
		blob_store-fsck.*verify blob store integrity
		blob_store-info-repo.*display blob store configuration
		blob_store-init.*initialize a local blob store
		blob_store-init-from.*initialize a blob store from a configuration file
		blob_store-init-pointer.*initialize a pointer blob store
		blob_store-install-mcp.*install MCP server configuration
		blob_store-init-sftp-explicit.*initialize an SFTP blob store with explicit credentials
		blob_store-init-inventory-archive.*initialize an inventory archive blob store
		blob_store-init-inventory-archive-v0.*initialize an inventory archive blob store \(v0\)
		blob_store-init-inventory-archive-v1.*initialize an inventory archive blob store \(v1\)
		blob_store-init-sftp-ssh_config.*initialize an SFTP blob store via ssh_config
		blob_store-list.*list configured blob stores
		blob_store-mcp.*start the MCP server
		blob_store-pack.*pack loose blobs into archive files
		blob_store-pack-cat-ids.*list blob digests contained in archive files
		blob_store-pack-list.*list archive files in inventory archive stores
		blob_store-pack-blobs.*write files and pack them into an archive
		blob_store-read.*read blobs from JSON on stdin
		blob_store-sync.*synchronize blobs between stores
		blob_store-write.*write blobs to a store
		cat-alfred.*output objects in Alfred workflow format
		cat-object.*output raw object content by markl id
		check-workspace.*check workspace state
		checkin.*commit workspace changes to the store
		checkin-blob.*commit blob changes with metadata updates
		checkin-json.*commit objects from JSON on stdin
		checkout.*check out objects to the workspace
		clean.*remove checked-out objects from the workspace
		clone.*clone a remote repository
		complete.*complete a command-line
		debug-print-probe-index.*print stream index probes
		deinit.*remove repository and workspace directories
		diff.*show differences between workspace and store
		dormant-add.*add tags to the dormant index
		dormant-edit.*edit dormant tags in an editor
		dormant-remove.*remove tags from the dormant index
		edit.*check out and edit objects in an editor
		edit-config.*edit the repository configuration
		exec.*execute a script stored as a blob
		export.*export objects to an inventory list archive
		find-missing.*find blob digests missing from stores
		format-blob.*format an object's blob content
		format-object.*format an object with a type formatter
		format-organize.*format an organize file
		fsck.*verify object integrity across stores
		gen.*generate cryptographic keys
		generate-zettel-id-components.*extract unique zettel id components from stdin
		import.*import objects from inventory list files
		info.*display repository information
		info-pivy_agent.*list ECDSA keys in pivy-agent
		info-ssh_agent.*list keys in the SSH agent
		info-repo.*display repository configuration
		info-workspace.*display workspace configuration
		init.*initialize a new repository
		init-workspace.*initialize a workspace directory
		install-mcp.*install MCP server configuration
		last.*display the most recently committed objects
		mcp.*start the MCP server
		merge-tool.*resolve merge conflicts with an external tool
		migrate-zettel-ids.*migrate zettel id flat files to log format
		new.*create new zettels
		organize.*organize objects with a text editor
		peek-zettel-ids.*preview available zettel ids
		pull.*pull objects from a remote repository
		pull-blob-store.*pull blobs from a remote blob store
		push.*push objects to a remote repository
		reindex.*rebuild store indices
		remote-add.*add a remote repository
		repo-fsck.*verify repository inventory list integrity
		revert.*revert objects to their stored state
		save.*commit workspace changes to the store
		serve.*start the HTTP server
		show.*display objects from the store
		status.*show workspace object state
		update.*update type lock signatures
		version.*print dodder build version and commit
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

function complete_blob_store_fsck { # @test
  run_dodder complete blob_store-fsck
  assert_success
  assert_output --regexp '\.default'
}

function complete_blob_store_sync { # @test
  run_dodder complete blob_store-sync
  assert_success
  assert_output --regexp '\.default'
}

function complete_repo_fsck { # @test
  run_dodder complete repo-fsck
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
