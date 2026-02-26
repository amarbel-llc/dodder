#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/common.bash"

  # for shellcheck SC2154
  export output

  copy_from_version "$DIR"

  run_dodder_init_workspace
}

teardown() {
  chflags_and_rm
}

# bats file_tags=user_story:organize

cmd_def_organize=(
  "${cmd_dodder_def[@]}"
  -prefix-joints=false
  -refine=true
)

cmd_def_organize_prefix_joints=(
  "${cmd_dodder_def[@]}"
  -prefix-joints=true
  -refine=true
)

function organize_empty { # @test
  run_dodder organize "${cmd_def_organize[@]}" -mode output-only
  assert_success
  assert_output_unsorted - <<-EOM
	EOM
}

function organize_empty_commit { # @test
  run_dodder organize "${cmd_def_organize[@]}" -mode commit-directly <<-EOM
		- test
	EOM

  assert_success
  assert_output - <<-EOM
		[two/uno !md "test"]
	EOM
}

function organize_simple { # @test
  actual="$(mktemp)"
  run_dodder organize "${cmd_def_organize[@]}" -mode output-only :z,e,t >"$actual"
  assert_success
  assert_output_unsorted - <<-EOM

		- [!md !toml-type-v1]
		- [one/dos !md tag-3 tag-4] wow ok again
		- [one/uno !md tag-3 tag-4] wow the first
	EOM
}

function organize_simple_commit { # @test
  run_dodder checkout one/uno
  assert_success

  run_dodder organize -mode commit-directly :z,e,t <<-EOM
		# new-etikett-for-all, %virtual_etikett
		- [   !md   ]
		- [   tag  ]
		- [   tag-1]
		- [   tag-2]
		- [   tag-3]
		- [   tag-4]
		- [one/dos   !md tag-3 tag-4] wow ok again
		- [one/uno   !md tag-3 tag-4] wow the first
	EOM
  assert_success
  assert_output_unsorted - <<-EOM
		[!md @$(get_type_blob_sha) !toml-type-v1 %virtual_etikett new-etikett-for-all]
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" %virtual_etikett new-etikett-for-all tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" %virtual_etikett new-etikett-for-all tag-3 tag-4]
		[tag-1 %virtual_etikett new-etikett-for-all]
		[tag-2 %virtual_etikett new-etikett-for-all]
		[tag-3 %virtual_etikett new-etikett-for-all]
		[tag-4 %virtual_etikett new-etikett-for-all]
		[tag %virtual_etikett new-etikett-for-all]
	EOM

  run_dodder show -format log new-etikett-for-all:z,e,t
  assert_success
  assert_output_unsorted - <<-EOM
		[!md @$(get_type_blob_sha) !toml-type-v1 new-etikett-for-all]
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" new-etikett-for-all tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" new-etikett-for-all tag-3 tag-4]
		[tag-1 new-etikett-for-all]
		[tag-2 new-etikett-for-all]
		[tag-3 new-etikett-for-all]
		[tag-4 new-etikett-for-all]
		[tag new-etikett-for-all]
	EOM
}

function organize_simple_checkedout_matchesmutter { # @test
  run_dodder checkout one/dos
  assert_success
  assert_output_unsorted - <<-EOM
		      checked out [one/dos.zettel @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
	EOM

  run_dodder organize -mode commit-directly :z,e,t <<-EOM
		# new-etikett-for-all
		- [   !md   ]
		- [   -tag  ]
		- [   -tag-1]
		- [   -tag-2]
		- [   -tag-3]
		- [   -tag-4]
		- [one/dos   !md tag-3 tag-4] wow ok again
		- [one/uno   !md tag-3 tag-4] wow the first
	EOM
  assert_success
  assert_output_unsorted - <<-EOM
		[!md @$(get_type_blob_sha) !toml-type-v1 new-etikett-for-all]
		[-tag-1 new-etikett-for-all]
		[-tag-2 new-etikett-for-all]
		[-tag-3 new-etikett-for-all]
		[-tag-4 new-etikett-for-all]
		[-tag new-etikett-for-all]
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" new-etikett-for-all tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" new-etikett-for-all tag-3 tag-4]
	EOM

  run_dodder show -format log new-etikett-for-all:z,e,t
  assert_success
  assert_output_unsorted - <<-EOM
		[!md @$(get_type_blob_sha) !toml-type-v1 new-etikett-for-all]
		[-tag-1 new-etikett-for-all]
		[-tag-2 new-etikett-for-all]
		[-tag-3 new-etikett-for-all]
		[-tag-4 new-etikett-for-all]
		[-tag new-etikett-for-all]
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" new-etikett-for-all tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" new-etikett-for-all tag-3 tag-4]
	EOM

  run_dodder status one/dos.zettel
  assert_success
  assert_output - <<-EOM
		             same [one/dos.zettel @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" new-etikett-for-all tag-3 tag-4]
	EOM
}

function organize_simple_checkedout_merge_no_conflict { # @test
  run_dodder checkout one/dos
  assert_success
  assert_output_unsorted - <<-EOM
		      checked out [one/dos.zettel @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
	EOM

  cat - >one/dos.zettel <<-EOM
		---
		# wow ok again
		- get_this_shit_merged
		- tag-3
		- tag-4
		! md
		---

		not another one, now with a different body
	EOM

  run_dodder organize -mode commit-directly :z,e,t <<-EOM
		# new-etikett-for-all
		- [   !md   ]
		- [one/dos   !md tag-3 tag-4] wow ok again
		- [one/uno   !md tag-3 tag-4] wow the first
	EOM
  assert_success
  assert_output_unsorted - <<-EOM
		[!md @$(get_type_blob_sha) !toml-type-v1 new-etikett-for-all]
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" new-etikett-for-all tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" new-etikett-for-all tag-3 tag-4]
	EOM

  run_dodder show -format log new-etikett-for-all:z,e,t
  assert_success
  assert_output_unsorted - <<-EOM
		[!md @$(get_type_blob_sha) !toml-type-v1 new-etikett-for-all]
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" new-etikett-for-all tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" new-etikett-for-all tag-3 tag-4]
	EOM

  run_dodder status one/dos.zettel
  assert_success
  assert_output - <<-EOM
		       conflicted [one/dos.zettel]
	EOM
  # assert_output - <<-EOM
  # 	          changed [one/dos.zettel @7ac3bdeb0ac8fd96cd7f8700a4bbc7a5d777fe26c50b52c20ecd726b255ec3d0 !md "wow ok again" get_this_shit_merged new-etikett-for-all tag-3 tag-4]
  # EOM
}

function organize_simple_checkedout_merge_conflict { # @test
  #TODO-project-2022-zit-collapse_skus
  cat - >txt.type <<-EOM
		---
		! toml-type-v1
		---

		binary = false
	EOM

  cat - >txt2.type <<-EOM
		---
		! toml-type-v1
		---

		binary = false
	EOM

  run_dodder checkin -delete .t
  assert_success
  assert_output_unsorted - <<-EOM
		          deleted [txt.type]
		          deleted [txt2.type]
		[!txt2 @blake2b256-qxzg22c3axe9m42tpwqd4usnfag4elp20q7zvnkgmyea4f4rwcwsurfp5e !toml-type-v1]
		[!txt @blake2b256-qxzg22c3axe9m42tpwqd4usnfag4elp20q7zvnkgmyea4f4rwcwsurfp5e !toml-type-v1]
	EOM

  run_dodder checkout one/dos
  assert_success
  assert_output_unsorted - <<-EOM
		      checked out [one/dos.zettel @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
	EOM

  cat - >one/dos.zettel <<-EOM
		---
		# wow ok again modified
		- get_this_shit_merged
		- tag-3
		- tag-4
		! txt
		---

		not another one, conflict time
	EOM

  run_dodder organize -mode commit-directly :z,e,t <<-EOM
		---
		- new-etikett-for-all
		---

		- [   !md   ]
		- [   -tag  ]
		- [   -tag-1]
		- [   -tag-2]
		- [   -tag-3]
		- [   -tag-4]
		- [one/dos   !txt2 tag-3 tag-4] wow ok again different
		- [one/uno   !txt2 tag-3 tag-4] wow the first
	EOM
  assert_success
  assert_output_unsorted - <<-EOM
		[!md @$(get_type_blob_sha) !toml-type-v1 new-etikett-for-all]
		[-tag new-etikett-for-all]
		[-tag-1 new-etikett-for-all]
		[-tag-2 new-etikett-for-all]
		[-tag-3 new-etikett-for-all]
		[-tag-4 new-etikett-for-all]
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !txt2 "wow ok again different" new-etikett-for-all tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !txt2 "wow the first" new-etikett-for-all tag-3 tag-4]
	EOM

  run_dodder show -format log new-etikett-for-all:z,e,t
  assert_success
  assert_output_unsorted - <<-EOM
		[!md @$(get_type_blob_sha) !toml-type-v1 new-etikett-for-all]
		[-tag-1 new-etikett-for-all]
		[-tag-2 new-etikett-for-all]
		[-tag-3 new-etikett-for-all]
		[-tag-4 new-etikett-for-all]
		[-tag new-etikett-for-all]
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !txt2 "wow ok again different" new-etikett-for-all tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !txt2 "wow the first" new-etikett-for-all tag-3 tag-4]
	EOM

  run_dodder status one/dos.zettel
  assert_success
  assert_output - <<-EOM
		       conflicted [one/dos.zettel]
	EOM
}

function organize_hides_hidden_tags_from_organize { # @test
  run_dodder dormant-add zz-archive
  assert_success
  assert_output ''

  to_add="$(mktemp)"
  {
    echo ---
    echo "# split hinweis for usability"
    echo - project-2021-dodder
    echo - zz-archive-task-done
    echo ! md
    echo ---
  } >"$to_add"

  run_dodder new -edit=false "$to_add"
  assert_success
  assert_output - <<-EOM
		[two/uno !md "split hinweis for usability" project-2021-dodder zz-archive-task-done]
	EOM

  run_dodder show two/uno
  assert_success
  assert_output - <<-EOM
		[two/uno !md "split hinweis for usability" project-2021-dodder zz-archive-task-done]
	EOM

  run_dodder organize -mode output-only project-2021-dodder:z
  assert_success
  assert_output - <<-EOM
		---
		- project-2021-dodder
		---
	EOM
}

function organize_dry_run { # @test
  expected_show="$(mktemp)"

  run_dodder show "${cmd_dodder_def[@]}" -format log :z,e,t
  expected_show="$output"

  run_dodder organize -dry-run -mode commit-directly :z,e,t <<-EOM
		# new-etikett-for-all
		- [   !md   ]
		- [   -tag  ]
		- [   -tag-1]
		- [   -tag-2]
		- [   -tag-3]
		- [   -tag-4]
		- [one/dos  ] wow ok again
		- [one/uno  ] wow the first
	EOM
  assert_success

  run_dodder show -format log :z,e,t
  assert_success
  assert_output_unsorted "$expected_show"
}

function organize_with_type_output { # @test
  run_dodder organize "${cmd_def_organize[@]}" -mode output-only !md:z
  assert_success
  assert_output - <<-EOM
		---
		! md
		---

		- [one/dos tag-3 tag-4] wow ok again
		- [one/uno tag-3 tag-4] wow the first
	EOM
}

function organize_with_type_commit { # @test
  run_dodder organize -mode commit-directly !md:z <<-EOM
		---
		! txt
		---

		- [one/dos tag-3 tag-4] wow ok again
		- [one/uno tag-3 tag-4] wow the first
	EOM

  assert_success
  assert_output_unsorted - <<-EOM
		[!txt !toml-type-v1]
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !txt "wow ok again" tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !txt "wow the first" tag-3 tag-4]
	EOM
}

function modify_description { # @test
  run_dodder organize -mode commit-directly :z,e,t <<-EOM

		- [   !md   ]
		- [   tag  ]
		- [   tag-1]
		- [   tag-2]
		- [   tag-3]
		- [   tag-4]
		- [one/dos   !md tag-3 tag-4] wow ok again was modified
		- [one/uno   !md tag-3 tag-4] wow the first was modified too
	EOM
  assert_success
  assert_output_unsorted - <<-EOM
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again was modified" tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first was modified too" tag-3 tag-4]
		[tag]
		[tag-1]
		[tag-2]
		[tag-3]
		[tag-4]
	EOM
}

function add_named { # @test
  # TODO modify organize to not require query group or else accidentally output unchanged objektes
  run_dodder organize -mode commit-directly :e <<-EOM
		# with-tag
		- [added_tag]
	EOM
  assert_success
  assert_output_unsorted - <<-EOM
		[added_tag with-tag]
	EOM
}

function organize_v5_outputs_organize_one_tag { # @test
  to_add="$(mktemp)"
  {
    echo "---"
    echo "# wow"
    echo "- ok"
    echo "! md"
    echo "---"
  } >"$to_add"

  run_dodder new -edit=false "$to_add"
  assert_success
  assert_output - <<-EOM
		[two/uno !md "wow" ok]
	EOM

  run_dodder show -format object-id o/u
  assert_success
  assert_output 'one/uno'

  run_dodder organize "${cmd_def_organize[@]}" -mode output-only ok
  assert_success
  assert_output - <<-EOM
		---
		- ok
		---

		- [two/uno !md] wow
	EOM
}

function organize_v5_outputs_organize_two_tags { # @test
  to_add="$(mktemp)"
  {
    echo "---"
    echo "# wow"
    echo "- ok"
    echo "- brown"
    echo "! md"
    echo "---"
  } >"$to_add"

  run_dodder new -edit=false "$to_add"
  assert_success
  assert_output_unsorted - <<-EOM
		[two/uno !md "wow" brown ok]
	EOM

  run_dodder organize "${cmd_def_organize[@]}" -mode output-only ok brown
  assert_success
  assert_output - <<-EOM
		---
		- brown
		- ok
		---

		- [two/uno !md] wow
	EOM

  run_dodder organize "${cmd_def_organize[@]}" \
    -mode commit-directly \
    ok brown <<-EOM
			      # ok

			- [two/uno !md] wow
		EOM

  assert_success
  assert_output - <<-EOM
		[two/uno !md "wow" ok]
	EOM

  run_dodder show -format text two/uno
  assert_success
  assert_output --regexp - <<-EOM
		---
		# wow
		- ok
		! md@.*
		---
	EOM
}

function organize_v5_outputs_organize_one_tags_group_by_one { # @test
  to_add="$(mktemp)"
  {
    echo "---"
    echo "# wow"
    echo "- task"
    echo "- priority-1"
    echo "- priority-2"
    echo "! md"
    echo "---"
  } >"$to_add"

  run_dodder new -edit=false "$to_add"
  assert_success
  assert_output - <<-EOM
		[two/uno !md "wow" priority-1 priority-2 task]
	EOM

  run_dodder organize "${cmd_def_organize[@]}" \
    -mode output-only \
    -group-by priority task
  assert_success
  assert_output - <<-EOM
		---
		- task
		---

		    # priority-1

		- [two/uno !md priority-2] wow

		    # priority-2

		- [two/uno !md priority-1] wow
	EOM

  return

  # shellcheck disable=2317
  run_dodder organize "${cmd_def_organize_prefix_joints[@]}" \
    -mode output-only \
    -group-by priority task

  # shellcheck disable=2317
  assert_success
  # shellcheck disable=2317
  assert_output - <<-EOM
		---
		- task
		---

		          # priority

		         ##         -1

		- [two/uno  !md] wow

		         ##         -2

		- [two/uno  !md] wow
	EOM
}

function organize_v5_outputs_organize_two_zettels_one_tags_group_by_one { # @test
  to_add="$(mktemp)"
  {
    echo "---"
    echo "# one/uno"
    echo "- task"
    echo "- priority-1"
    echo "! md"
    echo "---"
  } >"$to_add"

  run_dodder new -edit=false "$to_add"
  assert_success
  assert_output_unsorted - <<-EOM
		[two/uno !md "one/uno" priority-1 task]
	EOM

  to_add="$(mktemp)"
  {
    echo "---"
    echo "# two/dos"
    echo "- task"
    echo "- priority-2"
    echo "! md"
    echo "---"
  } >"$to_add"

  run_dodder new -edit=false "$to_add"
  assert_success
  assert_output - <<-EOM
		[one/tres !md "two/dos" priority-2 task]
	EOM

  # add prefix joints
  run_dodder organize "${cmd_def_organize[@]}" -mode output-only -group-by priority task
  assert_success
  assert_output - <<-EOM
		---
		- task
		---

		    # priority-1

		- [two/uno !md] one/uno

		    # priority-2

		- [one/tres !md] two/dos
	EOM
}

function organize_v5_commits_organize_one_tags_group_by_two { # @test
  to_add="$(mktemp)"
  {
    echo "---"
    echo "# one/uno"
    echo "- task"
    echo "- priority-1"
    echo "- w-2022-07-07"
    echo "! md"
    echo "---"
  } >"$to_add"

  run_dodder new -edit=false "$to_add"
  assert_success

  to_add="$(mktemp)"
  {
    echo "---"
    echo "# two/dos"
    echo "- task"
    echo "- priority-1"
    echo "- w-2022-07-06"
    echo "! md"
    echo "---"
  } >"$to_add"

  run_dodder new -edit=false "$to_add"
  assert_success

  to_add="$(mktemp)"
  {
    echo "---"
    echo "# 3"
    echo "- task"
    echo "- priority-1"
    echo "- w-2022-07-06"
    echo "! md"
    echo "---"
  } >"$to_add"

  run_dodder new -edit=false "$to_add"
  assert_success

  expected_organize="$(mktemp)"
  {
    echo "# task"
    echo
    echo "## priority-1"
    echo
    echo "### w-2022-07-06"
    echo
    echo "- [one/dos !md] two/dos"
    echo
    echo "## priority-2"
    echo
    echo "### w-2022-07-07"
    echo
    echo "- [one/uno !md] one/uno"
    echo
    echo "###"
    echo
    echo "- [two/uno !md] 3"
  } >"$expected_organize"

  run_dodder organize "${cmd_def_organize[@]}" -mode commit-directly -group-by priority,w task <"$expected_organize"
  assert_success

  to_add="$(mktemp)"
  {
    echo "---"
    echo "# one/uno"
    echo "- priority-2"
    echo "- task"
    echo "- w-2022-07-07"
    echo "! md"
    echo "---"
  } >"$to_add"

  run_dodder show -format text one/uno
  assert_success
  assert_output --regexp - <<-EOM
---
# one/uno
- priority-2
- task
- w-2022-07-07
! md@.*
---
EOM

  run_dodder show -format text two/uno
  assert_success
  assert_output --regexp - <<-EOM
---
# 3
- priority-2
- task
! md@.*
---
EOM
}

function organize_v5_commits_organize_one_tags_group_by_two_new_zettels { # @test
  to_add="$(mktemp)"
  {
    echo "---"
    echo "# one/uno"
    echo "- task"
    echo "- priority-1"
    echo "- w-2022-07-07"
    echo "! md"
    echo "---"
  } >"$to_add"

  run_dodder new -edit=false "$to_add"
  assert_success

  expected="$(mktemp)"
  {
    echo priority-1
    echo task
    echo w-2022-07-07
  } >"$expected"

  # run dodder cat -gattung hinweis
  # assert_output --partial "$(cat "$expected")"

  to_add="$(mktemp)"
  {
    echo "---"
    echo "# two/dos"
    echo "- task"
    echo "- priority-1"
    echo "- w-2022-07-06"
    echo "! md"
    echo "---"
  } >"$to_add"

  run_dodder new -edit=false "$to_add"
  assert_success

  {
    echo priority-1
    echo task
    echo w-2022-07-06
    echo w-2022-07-07
  } >"$expected"

  to_add="$(mktemp)"
  {
    echo "---"
    echo "# 3"
    echo "- task"
    echo "- priority-1"
    echo "- w-2022-07-06"
    echo "! md"
    echo "---"
  } >"$to_add"

  run_dodder new -edit=false "$to_add"
  assert_success

  expected_organize="$(mktemp)"
  {
    echo "# task"
    echo "- new zettel one"
    echo "## priority-1"
    echo "- new zettel two"
    echo "### w-2022-07-06"
    echo "- [one/dos !md] two/dos"
    echo "## priority-2"
    echo "### w-2022-07-07"
    echo "- [one/uno !md] one/uno"
    echo "###"
    echo "- new zettel three"
    echo "- [two/uno !md] 3"
  } >"$expected_organize"

  run_dodder organize \
    "${cmd_def_organize[@]}" \
    -mode commit-directly \
    -group-by priority,w \
    task <"$expected_organize"
  assert_success

  run_dodder show -format text one/uno
  assert_success
  assert_output --regexp - <<-EOM
---
# one/uno
- priority-2
- task
- w-2022-07-07
! md@.*
---
EOM

  run_dodder show -format text two/uno
  assert_success
  assert_output --regexp - <<-EOM
---
# 3
- priority-2
- task
! md@.*
---
EOM

  run_dodder show -format text one/tres
  assert_success

  run_dodder show -format text two/dos
  assert_success

  run_dodder show -format text three/uno
  assert_success

  {
    echo priority-1
    echo priority-2
    echo task
    echo w-2022-07-06
    echo w-2022-07-07
  } >"$expected"

  # TODO
  # run dodder cat-tags-schwanzen
  # assert_output "$(cat "$expected")"
}

function organize_v5_commits_no_changes { # @test
  one="$(mktemp)"
  {
    echo "---"
    echo "# one/uno"
    echo "- priority-1"
    echo "- task"
    echo "- w-2022-07-07"
    echo "! md"
    echo "---"
  } >"$one"

  run_dodder new -edit=false "$one"
  assert_success
  assert_output_unsorted - <<-EOM
		[two/uno !md "one/uno" priority-1 task w-2022-07-07]
	EOM

  two="$(mktemp)"
  {
    echo "---"
    echo "# two/dos"
    echo "- priority-1"
    echo "- task"
    echo "- w-2022-07-06"
    echo "! md"
    echo "---"
  } >"$two"

  run_dodder new -edit=false "$two"
  assert_success
  assert_output_unsorted - <<-EOM
		[one/tres !md "two/dos" priority-1 task w-2022-07-06]
	EOM

  three="$(mktemp)"
  {
    echo "---"
    echo "# 3"
    echo "- priority-1"
    echo "- task"
    echo "- w-2022-07-06"
    echo "! md"
    echo "---"
  } >"$three"

  run_dodder new -edit=false "$three"
  assert_success
  assert_output_unsorted - <<-EOM
		[two/dos !md "3" priority-1 task w-2022-07-06]
	EOM

  # TODO add prefix joints
  run_dodder organize "${cmd_def_organize[@]}" \
    -mode output-only \
    -group-by priority,w task
  assert_success
  assert_output - <<-EOM
		---
		- task
		---

		    # priority-1

		   ## w-2022-07-06

		- [one/tres !md] two/dos
		- [two/dos !md] 3

		   ## w-2022-07-07

		- [two/uno !md] one/uno

	EOM

  run_dodder organize "${cmd_def_organize[@]}" \
    -mode commit-directly \
    -group-by priority,w task \
    <<-EOM
			---
			- task
			---

			           # priority

			          ##         -1

			         ### w

			        ####  -2022-07

			       #####          -06

			- [two/uno   !md] one/uno

			       #####          -07

			- [one/tres  !md] two/dos
			- [two/dos   !md] 3

		EOM
  assert_success
  assert_output_unsorted - <<-EOM
		[one/tres !md "two/dos" priority-1 task w-2022-07-07]
		[two/dos !md "3" priority-1 task w-2022-07-07]
		[two/uno !md "one/uno" priority-1 task w-2022-07-06]
	EOM
}

function organize_v5_commits_dependent_leaf { # @test
  one="$(mktemp)"
  {
    echo "---"
    echo "# one/uno"
    echo "- priority-1"
    echo "- task"
    echo "- w-2022-07-07"
    echo "! md"
    echo "---"
  } >"$one"

  run_dodder new -edit=false "$one"
  assert_success

  two="$(mktemp)"
  {
    echo "---"
    echo "# two/dos"
    echo "- priority-1"
    echo "- task"
    echo "- w-2022-07-06"
    echo "! md"
    echo "---"
  } >"$two"

  run_dodder new -edit=false "$two"
  assert_success

  three="$(mktemp)"
  {
    echo "---"
    echo "# 3"
    echo "- priority-1"
    echo "- task"
    echo "- w-2022-07-06"
    echo "! md"
    echo "---"
  } >"$three"

  run_dodder new -edit=false "$three"
  assert_success

  expected_organize="$(mktemp)"
  {
    echo "# task"
    echo "## priority-2"
    echo "### w-2022-07"
    echo "#### -07"
    echo "- [one/dos !md] two/dos"
    echo "- [two/uno !md] 3"
    echo "#### -08"
    echo "- [one/uno !md] one/uno"
    echo "###"
  } >"$expected_organize"

  run_dodder organize "${cmd_def_organize[@]}" -verbose -mode commit-directly -group-by priority,w task <"$expected_organize"
  assert_success
}

function organize_v5_zettels_in_correct_places { # @test
  one="$(mktemp)"
  {
    echo "---"
    echo "# jabra coral usb_a-to-usb_c cable"
    echo "- inventory-pipe_shelves-atheist_shoes_box-jabra_yellow_box_2"
    echo "---"
  } >"$one"

  run_dodder new -edit=false "$one"

  run_dodder organize "${cmd_def_organize[@]}" \
    -mode output-only -group-by inventory \
    inventory-pipe_shelves-atheist_shoes_box-jabra_yellow_box_2
  assert_success

  # TODO add prefix joints
  assert_output - <<-EOM
		---
		- inventory-pipe_shelves-atheist_shoes_box-jabra_yellow_box_2
		---

		- [two/uno !md] jabra coral usb_a-to-usb_c cable
	EOM
}

function organize_v5_tags_correct { # @test

  run_dodder organize "${cmd_def_organize[@]}" -mode commit-directly <<-EOM
		# test1
		## -wow

		- zettel bez
	EOM
  assert_success

  assert_output - <<-EOM
		[two/uno !md "zettel bez" test1-wow]
	EOM

  mkdir -p one
  {
    echo "---"
    echo "- test4"
    echo "! md"
    echo "---"
  } >"one/uno.zettel"

  run_dodder checkin one/uno.zettel
  assert_success
  assert_output - <<-EOM
		[one/uno !md test4]
	EOM

  # TODO-P2 fix issue with kennung schwanzen
  # run_dodder cat-tags-schwanzen
  # assert_output - <<-EOM
  # EOM

  mkdir -p one
  {
    echo "---"
    echo "- test4"
    echo "- test1-ok"
    echo "! md"
    echo "---"
  } >"one/uno.zettel"

  run_dodder checkin one/uno.zettel
  assert_output - <<-EOM
		[one/uno !md test1-ok test4]
	EOM
}

function organize_remove_anchored_metadata { # @test
  run_dodder show tag-3:z
  assert_success
  assert_output_unsorted - <<-EOM
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM
  run_dodder organize "${cmd_def_organize[@]}" -mode commit-directly tag-3 <<-EOM
		---
		- tag-3
		---
	EOM

  assert_success
  assert_output_unsorted - <<-EOM
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-4]
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-4]
	EOM

  run_dodder show tag-3:z
  assert_success
  assert_output_unsorted - <<-EOM
	EOM
}

function organize_update_checkout { # @test
  run_dodder checkout one/dos
  assert_success
  assert_output_unsorted - <<-EOM
		      checked out [one/dos.zettel @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
	EOM

  run_dodder organize "${cmd_def_organize[@]}" -mode commit-directly :z <<-EOM
		---
		- test
		---

		- [one/dos  !md tag-3 tag-4] wow ok again
		- [one/uno  !md tag-3 tag-4] wow the first
	EOM
  assert_success
  assert_output_unsorted - <<-EOM
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4 test]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4 test]
	EOM

  run_dodder status
  assert_success
  assert_output_unsorted - <<-EOM
		             same [one/dos.zettel @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4 test]
	EOM
}

function organize_update_checkout_remove_tags { # @test
  run_dodder checkout one/dos
  assert_success
  assert_output_unsorted - <<-EOM
		      checked out [one/dos.zettel @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
	EOM

  run_dodder organize "${cmd_def_organize[@]}" -mode commit-directly :z <<-EOM
		- [one/dos  !md] wow ok again
		- [one/uno  !md] wow the first
	EOM
  assert_success
  assert_output_unsorted - <<-EOM
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again"]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first"]
	EOM

  run_dodder status
  assert_success
  assert_output_unsorted - <<-EOM
		             same [one/dos.zettel @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again"]
	EOM
}

function create_structured_zettels { # @test
  run_dodder organize "${cmd_def_organize[@]}" -mode commit-directly <<-EOM
		---
		- test
		---

		- [/] first
		- [/ !task tag-3] second
	EOM
  assert_success
  assert_output_unsorted - <<-EOM
		[!task !toml-type-v1]
		[one/tres !task "second" tag-3 test]
		[two/uno !md "first" test]
	EOM
}

function description_with_literal_characters { # @test
  run_dodder organize "${cmd_def_organize[@]}" -mode commit-directly <<-EOM
		- [terb/ala !md payee] thoughts on quincey's contract / scope of work
	EOM
  assert_success
  assert_output_unsorted - <<-EOM
		[terb/ala !md "thoughts on quincey's contract / scope of work" payee]
	EOM
}

# [hemp/mr !task project-2021-zit-bugs today zz-inbox] fix issue with `zit organize project-2021-zit` causing deltas
function tags_with_extended_tags_noop { # @test
  run_dodder organize -mode commit-directly :z <<-EOM
		# new-etikett-for-all
		- [one/dos   !md tag-3 tag-4] wow ok again
		- [one/uno   !md tag-3 tag-4] wow the first
	EOM
  assert_success
  assert_output_unsorted - <<-EOM
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" new-etikett-for-all tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" new-etikett-for-all tag-3 tag-4]
	EOM

  run_dodder organize -mode output-only new:z <<-EOM
		# new-etikett-for-all
		- [one/dos   !md tag-3 tag-4] wow ok again
		- [one/uno   !md tag-3 tag-4] wow the first
	EOM
  assert_success
  assert_output - <<-EOM
		---
		- new
		---

		- [one/dos !md new-etikett-for-all tag-3 tag-4] wow ok again
		- [one/uno !md new-etikett-for-all tag-3 tag-4] wow the first
	EOM

  run_dodder organize -mode commit-directly new:z <<-EOM
		# new

		- [one/dos !md new-etikett-for-all tag-3 tag-4] wow ok again
		- [one/uno !md new-etikett-for-all tag-3 tag-4] wow the first
	EOM
  assert_success
  assert_output ''
}

# bats test_tags=user_story:default_tags
function organize_new_objects_default_tags { # @test
  # shellcheck disable=SC2317
  function editor() (
    sed -i "s/tags = \\[]/tags = ['zz-inbox']/" "$0"
    # sed -i "/type = '!md'/a tags = 'hello'" "$0"
  )

  export -f editor

  export EDITOR="/bin/bash -c 'editor \$0'"
  run_dodder edit-config
  assert_success
  assert_output - <<-EOM
		[konfig @blake2b256-5y2xv5tx28jrkns3u8q4skx4xxjpfhzsl93njr2arpy5n8xsngwqkcay07 !toml-config-v2]
	EOM

  run_dodder organize -mode output-only
  assert_success
  assert_output - <<-EOM
		---
		- zz-inbox
		---
	EOM

  # shellcheck disable=SC2317
  function editor() (
    echo "- new zettel object" >"$0"
  )

  run_dodder organize
  assert_success
  assert_output - <<-EOM
		[two/uno !md "new zettel object"]
	EOM

  # shellcheck disable=SC2317
  function editor() (
    echo "- new zettel object" >>"$0"
  )

  run_dodder organize
  assert_success
  assert_output - <<-EOM
		[one/tres !md "new zettel object" zz-inbox]
	EOM
}

# [nob/golb !task project-2021-zit-bugs project-2021-zit-v1 today zz-inbox] fix issue with newlines rendered in organize
function object_with_newline_in_description { # @test
  run_dodder new -edit=false - <<-EOM
		---
		# description that has
		# newline
		---
	EOM
  assert_success
  assert_output_unsorted - <<-EOM
		[two/uno !md "description that has newline"]
	EOM
}

function organize_checked_out { # @test
  run_dodder checkout :z,e,t
  assert_success
  assert_output_unsorted - <<-EOM
		      checked out [md.type @blake2b256-3kj7xgch6rjkq64aa36pnjtn9mdnl89k8pdhtlh33cjfpzy8ek4qnufx0m !toml-type-v1]
		      checked out [one/dos.zettel @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		      checked out [one/uno.zettel @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM

  run_dodder organize -mode output-only .
  assert_success
  assert_output - <<-EOM

		- [md.type !toml-type-v1]
		- [one/dos.zettel !md tag-3 tag-4] wow ok again
		- [one/uno.zettel !md tag-3 tag-4] wow the first
	EOM
}

# bats test_tags=user_story:fs_blobs, user_story:external_ids
function organize_output_only_fs_blobs() { # @test
  cat >test.md <<-EOM
		newest body
	EOM

  run_dodder organize -mode output-only .
  assert_success
  assert_output - <<-EOM

		- [test.md]
	EOM
}

# bats test_tags=user_story:fs_blobs, user_story:organize, user_story:external_ids
function organize_untracked_fs_blob_with_spaces() { # @test
  cat >"test with spaces.txt" <<-EOM
		newest body
	EOM

  run_dodder organize -mode output-only "test with spaces.txt"
  assert_success
  assert_output_unsorted - <<-EOM

		- ["test with spaces.txt"]
	EOM
}

# bats test_tags=user_story:organize,user_story:workspace,user_story:default_tags
function organize_default_tags_workspace { # @test
  # shellcheck disable=SC2317
  function editor() (
    sed -i "s/tags = \\[]/tags = ['zz-inbox']/" "$0"
    # sed -i "/type = '!md'/a tags = 'hello'" "$0"
  )

  export -f editor

  export EDITOR="/bin/bash -c 'editor \$0'"
  run_dodder edit-config
  assert_success
  assert_output - <<-EOM
		[konfig @blake2b256-5y2xv5tx28jrkns3u8q4skx4xxjpfhzsl93njr2arpy5n8xsngwqkcay07 !toml-config-v2]
	EOM

  cat >.dodder-workspace <<-EOM
		---
		! toml-workspace_config-v0
		---

		query = "today"
	EOM

  run_dodder info-workspace query
  assert_success
  assert_output 'today'

  run_dodder new -edit=false - <<-EOM
		---
		# test default tags
		- tag-3
		- today
		- zz-inbox
		! md
		---

		body
	EOM
  assert_success
  assert_output_unsorted - <<-EOM
		[two/uno @blake2b256-gu738nunyrnsqukgqkuaau9zslu0fhwg4dgs9ltuyvnlp42wal8sdpn2hc !md "test default tags" tag-3 today zz-inbox]
	EOM

  actual="$(mktemp)"
  run_dodder organize "${cmd_def_organize[@]}" -mode output-only -group-by tag :z,e,t >"$actual"
  assert_success
  assert_output - <<-EOM
		---
		- today
		---

		    # tag-3

		- [two/uno !md zz-inbox] test default tags
	EOM
}

# bats test_tags=user_story:organize,user_story:workspace
function organize_dot_operator_workspace_delete_files { # @test
  skip
  # shellcheck disable=SC2317
  function editor() (
    sed -i "s/tags = \\[]/tags = ['zz-inbox']/" "$0"
    # sed -i "/type = '!md'/a tags = 'hello'" "$0"
  )

  export -f editor

  export EDITOR="/bin/bash -c 'editor \$0'"
  run_dodder edit-config
  assert_success
  assert_output - <<-EOM
		[konfig @920a6a8fe55112968d75a2c77961a311343cfd62cdcc2305aff913afee7fa638 !toml-config-v2]
	EOM

  cat >.dodder-workspace <<-EOM
		---
		! toml-workspace_config-v0
		---

		query = "today"
	EOM

  run_dodder info-workspace query
  assert_success
  assert_output 'today'

  echo "file one" >1.md
  echo "file two" >2.md

  function editor() {
    # shellcheck disable=SC2317
    cat - >"$1" <<-EOM
			---
			- today
			---

			- ["1.md"]
			- ["2.md"]
		EOM
  }

  export -f editor

  # shellcheck disable=SC2016
  export EDITOR='bash -c "editor $0"'

  run_dodder organize .
  assert_success
  assert_output - <<-EOM
		[two/uno @blake2b256-5hwedpxxtvucp2wnhcwafgt6y0a93qca3x0522x2j6kmlw0zzp9qvmvt2s !md "1" tag-3 tag-two]
		[one/tres @blake2b256-ax76uj5gxlkxj0za603p78t3fzyl23tzd977js8qkzv3j5lx8v9smrj5ch !md "2" tag-3 tag-one]
		          deleted [1.md]
		          deleted [2.md]
	EOM
}
