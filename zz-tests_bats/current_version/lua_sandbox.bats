#! /usr/bin/env bats

# End-to-end tests for the Lua VM pool sandbox (issue #389).
# Verifies that the sandbox blocks io/os access and that committed hook blobs
# which reach for sandboxed globals fail loudly at execution time.

setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

  # for shellcheck SC2154
  export output

  run_dodder_init_disable_age
}

teardown() {
  chflags_nouchg
}

# A type whose on_commit_fields hook calls io.open must fail at commit time with
# an actionable error message rather than a generic "nil value" panic.
function lua_sandbox_blocks_io_access { # @test
  cat - >io-probe.type <<-'EOM'
		file-extension = "toml"
		vim-syntax-type = "toml"
		hooks = """
		return {
		  on_commit_fields = function(kinder, mutter)
		    io.open("/tmp/dodder-sandbox-probe", "w")
		  end,
		}
		"""
	EOM

  run_dodder checkin -delete io-probe.type
  assert_success

  run_dodder new -edit=false - <<-EOM
		---
		# sandbox probe
		! io-probe
		---

	EOM
  assert_failure
  assert_output --regexp 'io is not available in dodder Lua scripts'
}

# A type whose on_commit_fields hook calls os.date must fail at commit time
# with an actionable error message pointing to dodder_today().
function lua_sandbox_blocks_os_access { # @test
  cat - >os-probe.type <<-'EOM'
		file-extension = "toml"
		vim-syntax-type = "toml"
		hooks = """
		return {
		  on_commit_fields = function(kinder, mutter)
		    os.date("!%Y-%m-%d")
		  end,
		}
		"""
	EOM

  run_dodder checkin -delete os-probe.type
  assert_success

  run_dodder new -edit=false - <<-EOM
		---
		# sandbox probe
		! os-probe
		---

	EOM
  assert_failure
  assert_output --regexp 'dodder_today'
}
