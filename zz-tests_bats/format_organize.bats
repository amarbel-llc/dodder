#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/common.bash"

  # for shellcheck SC2154
  export output
}

teardown() {
  chflags_nouchg
}

# bats file_tags=user_story:organize

function format_organize_right_align { # @test
  wd="$(mktemp -d)"
  cd "$wd" || exit 1

  run_dodder_init_disable_age
  assert_success

  to_add="$(mktemp)"
  cat - >"$to_add" <<-EOM
		# task
		## urgency
		### urgency-1
		- [!md]
		- [-zz-archive]
		### -2
	EOM

  expected="$(mktemp)"
  cat - >"$expected" <<-EOM

		    # task

		   ## urgency

		  ###        -1

		- [!md]
		- [-zz-archive]

		  ###        -2
	EOM

  run_dodder format-organize -prefix-joints=true -refine=true "$to_add"
  assert_success
  assert_output "$(cat "$expected")"
}

# bats user_story:organize
function format_organize_left_align { # @test
  cd "$BATS_TEST_TMPDIR" || exit 1
  run_dodder_init_disable_age

  to_add="$(mktemp)"
  cat - >"$to_add" <<-EOM
		# task
		## urgency
		### urgency-1
		### -2
	EOM

  expected="$(mktemp)"
  cat - >"$expected" <<-EOM

		    # task

		   ## urgency

		  ###        -1

		  ###        -2
	EOM

  run_dodder format-organize -prefix-joints=true -refine "$to_add"
  assert_success
  assert_output "$(cat "$expected")"
}

cmd_def_organize=(
  -prefix-joints=true
  -refine=true
)

cat_organize() (
  cat - <<-EOM

		- [ach/vil] blah

		     # %project

		    ##         -2021-dodder

		   ###                     -22q1-uws-140

		  ####                                  -moving

		- [io/poliwr] update billing addresses

		  ####                                  -mvp-main_room

		- [prot/nidora] Brainstorm where to place toolbox.md

		   ###                     -commands

		- [mer/golb] use error types to generate specific exit status codes
		- [tec/slowp] update output of commands to use new store

		   ###                     -etiketten_and_organize

		- [pe/mo] add etikett rule type for removing etiketts based on conditions
		- [yttr/gole] use default etiketten with add

		   ###                     -init

		- [ph/hitmonc] Add bats test for initing more than once.md
		- [rub/rap] add .exrc to init
	EOM
)

function outputs_organize_one_tag { # @test
  cd "$BATS_TEST_TMPDIR" || exit 1
  run_dodder_init_disable_age

  run_dodder format-organize "${cmd_def_organize[@]}" <(cat_organize)
  assert_output "$(cat_organize)"
}

function format_organize_create_structured_zettels { # @test
  run_dodder_init_disable_age

  function cat_body {
    cat <<-EOM
			---
			- test
			---

			- [/] first
			- [/ !task tag-3] second
			- third
		EOM
  }

  run_dodder format-organize "${cmd_def_organize[@]}" <(cat_body)
  assert_success
  assert_output - <<-EOM
		---
		- test
		---

		- [/] first
		- [/ !task tag-3] second
		- [/] third
	EOM
}

function format_organize_create_bare_object_description_line_wrap { # @test
  run_dodder_init_disable_age

  function cat_body {
    cat <<-EOM
			---
			- test
			---

			- this is a long
			  description
		EOM
  }

  run_dodder format-organize "${cmd_def_organize[@]}" <(cat_body)
  assert_success
  assert_output - <<-EOM
		---
		- test
		---

		- [/] this is a long description
	EOM
}

# bats test_tags=user_story:external_ids
function format_organize_with_fields_and_instructions { # @test
  run_dodder_init_disable_age

  function cat_body {
    cat <<-EOM
			---
			% instructions: to prevent an object from being checked in, delete it entirely
			% delete:false delete once checked in
			---

			- [/firefox-ddog/bookmark-9ikbbKaAXmb7 title="CI Visibility Tests" url="https://docs.datadoghq.com/api/latest/ci-visibility-tests/#aggregate-tests-events"] CI Visibility Tests
			- [/firefox-ddog/bookmark-BNLiQZXO8rEK title="Unbound can't resolve specific domain : r/pihole" url="https://www.reddit.com/r/pihole/comments/os2jqb/unbound_cant_resolve_specific_domain/"] Unbound can't resolve specific domain : r/pihole
		EOM
  }

  run_dodder format-organize "${cmd_def_organize[@]}" <(cat_body)
  assert_success
  assert_output "$(cat_body)"
}

# bats test_tags=user_story:external_ids
function format_organize_untracked_fs_blob_with_spaces() { # @test
  run_dodder_init_disable_age
  run_dodder format-organize - <<-EOM

		- ["test with spaces.txt"]
	EOM
  assert_success
  assert_output_unsorted - <<-EOM

		- ["test with spaces.txt"]
	EOM
}

# bats test_tags=user_story:external_ids
# TODO [anti/deb !task zz-inbox] fix `dodder organize .`
function format_organize_recognized_fs_blob_with_newlines() { # @test
  skip
  run_dodder_init_disable_age
  run_dodder format-organize - <<-EOM
		- [one/uno !pdf payee-x-heloc zz-inbox
		                   "heloc-board/CO-OP Modified Clarity Comittment - letterhead.pdf"] CO-OP Modified Clarity Comittment - letterhead
		- [two/dos !pdf area-money-tax
		                                  heloc-board/2022 taxes.pdf] us tax return
	EOM
  assert_success
  assert_output_unsorted - <<-EOM

		- [americium/bartok !pdf payee-bank_of_america-heloc project-24q1-reno-heloc-board_approval-docs zz-inbox] heloc-board/CO-OP Modified Clarity Comittment - BOA letterhead.pdf CO-OP Modified Clarity Comittment - BOA letterhead
	EOM
}

function format_organize_with_new_spreading_several_lines { # @test
  run_dodder_init_disable_age

  function cat_body {
    cat <<-EOM
			---
			- today
			---

			- [abo/gal !task pom-4 priority-2_want zz-inbox] john jacob
			- [mes/mare !task pom-4 priority-2_want today-in_progress zz-inbox] john jacob jingleheimer smith
			- [ne/har !task pom-1 priority-1_should zz-inbox] jingleheimer smith

			- [/ !task pom-1] john jacob jingleheimer smith
			that's my name too
		EOM
  }

  run_dodder format-organize "${cmd_def_organize[@]}" <(cat_body)
  assert_success
  assert_output - <<-EOM
		---
		- today
		---

		- [/ !task pom-1] john jacob jingleheimer smith that's my name too
		- [abo/gal !task pom-4 priority-2_want zz-inbox] john jacob
		- [mes/mare !task pom-4 priority-2_want today-in_progress zz-inbox] john jacob jingleheimer smith
		- [ne/har !task pom-1 priority-1_should zz-inbox] jingleheimer smith
	EOM
}

function format_organize_with_new_spreading_several_lines_and_ambiguous_heading { # @test
  run_dodder_init_disable_age

  function cat_body {
    cat <<-EOM
			---
			- today
			---

			- [abo/gal !task pom-4 priority-2_want zz-inbox] john jacob
			- [mes/mare !task pom-4 priority-2_want today-in_progress zz-inbox] john jacob jingleheimer smith
			- [ne/har !task pom-1 priority-1_should zz-inbox] jingleheimer smith
			# ambiguous

			- [/ !task pom-1] john jacob jingleheimer smith
			that's my name too
		EOM
  }

  run_dodder format-organize "${cmd_def_organize[@]}" <(cat_body)
  assert_success
  assert_output - <<-EOM
		---
		- today
		---

		- [abo/gal !task pom-4 priority-2_want zz-inbox] john jacob
		- [mes/mare !task pom-4 priority-2_want today-in_progress zz-inbox] john jacob jingleheimer smith
		- [ne/har !task pom-1 priority-1_should zz-inbox] jingleheimer smith

		    # ambiguous

		- [/ !task pom-1] john jacob jingleheimer smith that's my name too
	EOM
}

function format_organize_with_heading_having_space { # @test
  run_dodder_init_disable_age

  function cat_body {
    cat <<-EOM
			---
			- today
			---

			- [abo/gal !task pom-4 priority-2_want zz-inbox] john jacob
			- [mes/mare !task pom-4 priority-2_want today-in_progress zz-inbox] john jacob jingleheimer smith
			- [ne/har !task pom-1 priority-1_should zz-inbox] jingleheimer smith
			# 

			- [/ !task pom-1] john jacob jingleheimer smith
			that's my name too
		EOM
  }

  run_dodder format-organize "${cmd_def_organize[@]}" <(cat_body)
  assert_success
  assert_output - <<-EOM
		---
		- today
		---

		- [/ !task pom-1] john jacob jingleheimer smith that's my name too
		- [abo/gal !task pom-4 priority-2_want zz-inbox] john jacob
		- [mes/mare !task pom-4 priority-2_want today-in_progress zz-inbox] john jacob jingleheimer smith
		- [ne/har !task pom-1 priority-1_should zz-inbox] jingleheimer smith
	EOM
}
