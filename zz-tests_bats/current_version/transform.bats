#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

  # for shellcheck SC2154
  export output

  setup_repo
}

teardown() {
  teardown_repo
}

# bats file_tags=user_story:transform

# Fixture state (from create_test_zettels):
#   one/uno: !md "wow the first" tag-3 tag-4, body "last time"
#   one/dos: !md "wow ok again" tag-3 tag-4, body "not another one"
# The default query (latest+hidden zettels) expands to those two plus the
# !md type object via the transitive closure, so 3 objects are selected.

function transform_requires_script { # @test
  run_dodder transform
  assert_failure
  assert_output --regexp 'one of -script or -script-digest is required'
}

function transform_script_and_digest_mutually_exclusive { # @test
  echo 'return dodder.list()' >t.lua

  run_dodder transform -script t.lua -script-digest blake2b256-2qwngrkkpcptsnphu6jcyrwmtpyxux0hmsg4pjfpsn0tr7yt732sgk5lza
  assert_failure
  assert_output --regexp -- '-script and -script-digest are mutually exclusive'
}

function transform_script_must_return_handle { # @test
  # a non-table return is rejected by the VM pool's own chunk validation,
  # before the transform's handle check can run
  echo 'return 42' >t.lua

  run_dodder transform -script t.lua
  assert_failure
  assert_output --regexp 'expected table but got number'

  # a table that is not the dodder.list() handle reaches the handle check
  echo 'return {}' >t.lua

  run_dodder transform -script t.lua
  assert_failure
  assert_output --regexp 'script must return the dodder.list\(\) handle'
}

function transform_dry_run_does_not_commit { # @test
  cat >t.lua <<-'EOM'
		local list = dodder.list()

		for object in list:each() do
		  object.Etiketten["tag-4"] = nil
		end

		return list
	EOM

  run_dodder transform -dry_run -script t.lua
  assert_success
  # the TAP version header is the quiet validation pass announcing itself
  assert_output - <<-EOM
		selected 3 object(s)
		TAP version 14
		plan: 3 entries
		  import: 3
		dry run: not committed
	EOM

  run_dodder show :?z
  assert_success
  assert_output_unsorted - <<-EOM
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM
}

function transform_tag_cleanup_commits { # @test
  cat >t.lua <<-'EOM'
		local list = dodder.list()

		for object in list:each() do
		  object.Etiketten["tag-4"] = nil
		end

		return list
	EOM

  run_dodder transform -script t.lua
  assert_success

  run_dodder show :z
  assert_success
  assert_output_unsorted - <<-EOM
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3]
	EOM
}

function transform_type_rewrite_commits { # @test
  cat >t.lua <<-'EOM'
		local list = dodder.list()

		for object in list:each() do
		  if object.Gattung == "Zettel" then
		    object.Typ = "md2"
		  end
		end

		return list
	EOM

  run_dodder transform -script t.lua
  assert_success

  run_dodder show :z
  assert_success
  assert_output_unsorted - <<-EOM
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md2 "wow ok again" tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md2 "wow the first" tag-3 tag-4]
	EOM
}

function transform_remove_leaves_object_untouched { # @test
  cat >t.lua <<-'EOM'
		local list = dodder.list()

		for object in list:each() do
		  if object.Kennung == "one/uno" then
		    object.Etiketten["extra"] = true
		  else
		    list:remove(object)
		  end
		end

		return list
	EOM

  run_dodder transform -script t.lua
  assert_success

  run_dodder show :z
  assert_success
  assert_output_unsorted - <<-EOM
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" extra tag-3 tag-4]
	EOM
}

function transform_add_creates_zettel { # @test
  cat >t.lua <<-'EOM'
		local list = dodder.list()

		local fresh = list:add()
		fresh.Typ = "md"
		fresh.Etiketten["brand-new"] = true

		return list
	EOM

  run_dodder transform -script t.lua
  assert_success

  run_dodder show brand-new:z
  assert_success
  assert_output - <<-EOM
		[two/uno !md brand-new]
	EOM
}

function transform_no_new_objects_rejects_add { # @test
  cat >t.lua <<-'EOM'
		local list = dodder.list()

		local fresh = list:add()
		fresh.Typ = "md"

		return list
	EOM

  run_dodder transform -no_new_objects -script t.lua
  assert_failure
  assert_output --regexp 'is not present in the input list'

  run_dodder show :?z
  assert_success
  assert_output_unsorted - <<-EOM
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM
}

function transform_validation_catches_dangling_blob { # @test
  # a valid digest whose content is absent from this repo's stores (it
  # belongs to checkin_blob.bats' "the body" content)
  cat >t.lua <<-'EOM'
		local list = dodder.list()

		for object in list:each() do
		  if object.Kennung == "one/uno" then
		    object.Blob = "blake2b256-vl6ghtv2jsxppshflt86ardlx55ctn8jswx8j59tnv8r99uhs63syxsruy"
		  end
		end

		return list
	EOM

  run_dodder transform -script t.lua
  assert_failure
  assert_output --regexp 'not ok'
  assert_output --regexp 'transform output failed validation: 1 error'

  run_dodder show one/uno
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM

  # a deliberately staged inconsistent pass is allowed to skip validation
  run_dodder transform -skip_validation -script t.lua
  assert_success

  run_dodder show one/uno
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-vl6ghtv2jsxppshflt86ardlx55ctn8jswx8j59tnv8r99uhs63syxsruy !md "wow the first" tag-3 tag-4]
	EOM
}

function transform_blob_ffi_roundtrip { # @test
  cat >t.lua <<-'EOM'
		local list = dodder.list()

		for object in list:each() do
		  if object.Kennung == "one/uno" then
		    local bytes = blobs.read(object.Blob)
		    object.Blob = blobs.write(bytes .. "and even more\n")
		  end
		end

		return list
	EOM

  run_dodder transform -script t.lua
  assert_success

  run_dodder show -format blob one/uno
  assert_success
  assert_output - <<-EOM
		last time
		and even more
	EOM
}

function transform_script_digest_loads_stored_script { # @test
  cat >t.lua <<-'EOM'
		local list = dodder.list()

		for object in list:each() do
		  object.Etiketten["from-stored-script"] = true
		end

		return list
	EOM

  run_madder write -format tap t.lua
  assert_success

  digest="$(echo "$output" | grep -o 'blake2b256-[a-z0-9]*' | head -n1)"

  run_dodder transform -script-digest "$digest"
  assert_success

  run_dodder show from-stored-script:z
  assert_success
  assert_output_unsorted - <<-EOM
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" from-stored-script tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" from-stored-script tag-3 tag-4]
	EOM
}
