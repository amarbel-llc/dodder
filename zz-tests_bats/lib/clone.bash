# Helpers shared between clone.bats (direct-transfer / stdio variants)
# and clone_port.bats (TCP/HTTP variant via -handshake harness).

function bootstrap {
  mkdir -p "$1"
  (
    pushd "$1" || exit 1
    run_dodder_init -repo_id . "test-repo-id-them"

    {
      echo "---"
      echo "# wow"
      echo "- tag"
      echo "! md"
      echo "---"
      echo
      echo "body"
    } >to_add

    run_dodder new -edit=false to_add
    assert_success
    assert_output - <<-EOM
			[one/uno @blake2b256-gu738nunyrnsqukgqkuaau9zslu0fhwg4dgs9ltuyvnlp42wal8sdpn2hc !md "wow" tag]
		EOM

    run_dodder new -edit=false - <<-EOM
			---
			# zettel with multiple etiketten
			- this_is_the_first
			- this_is_the_second
			! md
			---

			zettel with multiple etiketten body
		EOM

    assert_success
    assert_output - <<-EOM
			[one/dos @blake2b256-fm7kce7793j3npevpm29spk04r6ycxv38dvx3hjxlzl8tcm5m3qq2mml86 !md "zettel with multiple etiketten" this_is_the_first this_is_the_second]
		EOM
  )
}

function run_clone_default_with() {
  run_dodder clone \
    -encryption none \
    -yin <(cat_yin) \
    -yang <(cat_yang) \
    -repo_id . \
    "$@"
}

function try_add_new_after_clone {
  run_dodder init-workspace -experimental-repo=false
  assert_success

  run_dodder new -edit=false - <<-EOM
		---
		# zettel after clone description
		! md
		---

		zettel after clone body
	EOM

  assert_success
  assert_output - <<-EOM
		[two/uno @blake2b256-kn7w3q7c3xvfa2p78wny0h79f7hd72nxtded0gvymu33wcnr2qmscl46ar !md "zettel after clone description"]
	EOM
}
