#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"
  export output

  # dodder serve loads its blob stores from a madder store; each repo gets its
  # own XDG home so the transfer genuinely moves objects over the wire. Each
  # test exports DODDER/MADDER_XDG_UTILITY_OVERRIDE to one of these before
  # operating on that repo. start_server's coproc inherits the override that is
  # exported at fork time (so it keeps serving `them` after the parent switches
  # to us_home).
  export them_home="$BATS_TEST_TMPDIR/them-home"
  export us_home="$BATS_TEST_TMPDIR/us-home"
}

teardown() {
  stop_server
  chflags_nouchg
}

# bats file_tags=serve,user_story:pull,user_story:push,user_story:remote

# These exercise merge resolution over the HTTP/stdio (remote_http) TCP
# transport, the "stage 4" companion to clone_port.bats's clone coverage. Pull
# imports locally (ParentNegotiatorFirstAncestor over GET /object-history); push
# imports server-side, where the client ships full object history in the POSTed
# inventory list and the server builds an in-band negotiator from it (#299).

# new_one_uno authors the shared base zettel one/uno in the current repo.
new_one_uno() {
  run_dodder new -edit=false - <<-EOM
		---
		# wow
		- tag
		! md
		---

		body
	EOM
  assert_success
}

# edit_one_uno DESC BODY checks out one/uno, rewrites it, and checks it back in.
edit_one_uno() {
  local desc="$1"
  local body="$2"

  run_dodder checkout one/uno
  assert_success

  {
    echo "---"
    echo "# ${desc}"
    echo "- tag"
    echo "! md"
    echo "---"
    echo
    echo "${body}"
  } >one/uno.zettel

  run_dodder checkin -delete one/uno.zettel
  assert_success
}

# push_over_http_fast_forward: us pushes one/uno v0 to them, edits it into a
# strict linear descendant, and pushes again. The second push must fast-forward
# on them (the receiver/server), not raise a spurious conflict (#299). The
# server stays up the whole test — us mutates only its own store between pushes.
function push_over_http_fast_forward { # @test
  export DODDER_XDG_UTILITY_OVERRIDE="$them_home"
  export MADDER_XDG_UTILITY_OVERRIDE="$them_home"

  mkdir -p them
  (
    pushd them || exit 1
    run_dodder_init
  )

  start_server them -public

  export DODDER_XDG_UTILITY_OVERRIDE="$us_home"
  export MADDER_XDG_UTILITY_OVERRIDE="$us_home"

  mkdir -p us
  pushd us || exit 1

  run_dodder_init
  new_one_uno

  run_dodder remote-add \
    toml-repo-uri-v0 \
    "http://${server_addr}" \
    them-http
  assert_success

  # First push: them receives one/uno v0.
  run_dodder push /them-http one/uno
  assert_success

  edit_one_uno "wow the second" "edited locally"

  # Second push must fast-forward on them, not raise a spurious conflict.
  run_dodder push /them-http one/uno
  assert_success
  popd || exit 1

  # them holds the descendant.
  stop_server

  export DODDER_XDG_UTILITY_OVERRIDE="$them_home"
  export MADDER_XDG_UTILITY_OVERRIDE="$them_home"

  pushd them || exit 1
  run_dodder show one/uno
  assert_success
  assert_output --regexp '^\[one/uno @blake2b256-.+ !md "wow the second" tag\]$'
  popd || exit 1
}

# push_over_http_divergence_conflict is the discriminator for the HTTP push
# negotiator (#299): us pushes v0, then us and them edit one/uno independently
# off that shared base. us's second push must fail with a conflict on them (the
# server returns 500), not silently overwrite them's divergent version.
function push_over_http_divergence_conflict { # @test
  export DODDER_XDG_UTILITY_OVERRIDE="$them_home"
  export MADDER_XDG_UTILITY_OVERRIDE="$them_home"

  mkdir -p them
  (
    pushd them || exit 1
    run_dodder_init
  )

  start_server them -public

  export DODDER_XDG_UTILITY_OVERRIDE="$us_home"
  export MADDER_XDG_UTILITY_OVERRIDE="$us_home"

  mkdir -p us
  pushd us || exit 1

  run_dodder_init
  new_one_uno

  run_dodder remote-add \
    toml-repo-uri-v0 \
    "http://${server_addr}" \
    them-http
  assert_success

  # Push the shared v0 to them.
  run_dodder push /them-http one/uno
  assert_success
  popd || exit 1

  # them diverges off v0. Stop the server so them's store is free to mutate,
  # then restart (new OS-assigned port).
  stop_server

  export DODDER_XDG_UTILITY_OVERRIDE="$them_home"
  export MADDER_XDG_UTILITY_OVERRIDE="$them_home"

  pushd them || exit 1
  edit_one_uno "wow the remote" "edited remotely"
  popd || exit 1

  start_server them -public

  # us diverges independently off the same v0.
  export DODDER_XDG_UTILITY_OVERRIDE="$us_home"
  export MADDER_XDG_UTILITY_OVERRIDE="$us_home"

  pushd us || exit 1
  edit_one_uno "wow the local" "edited locally"

  run_dodder remote-add \
    toml-repo-uri-v0 \
    "http://${server_addr}" \
    them-http2
  assert_success

  # Genuine divergence: the push must fail, not silently overwrite them's edit.
  run_dodder push /them-http2 one/uno
  assert_failure
  popd || exit 1

  # them still holds its own divergent version — us's push did not clobber it.
  stop_server

  export DODDER_XDG_UTILITY_OVERRIDE="$them_home"
  export MADDER_XDG_UTILITY_OVERRIDE="$them_home"

  pushd them || exit 1
  run_dodder show one/uno
  assert_success
  assert_output --regexp '^\[one/uno @blake2b256-.+ !md "wow the remote" tag\]$'
  popd || exit 1
}

# pull_over_http_fast_forward: us pulls one/uno v0, then them edits it into a
# strict linear descendant, then us pulls again and must fast-forward (no
# spurious conflict). Coverage for the already-implemented HTTP pull merge path.
function pull_over_http_fast_forward { # @test
  export DODDER_XDG_UTILITY_OVERRIDE="$them_home"
  export MADDER_XDG_UTILITY_OVERRIDE="$them_home"

  mkdir -p them
  (
    pushd them || exit 1
    run_dodder_init
    new_one_uno
  )

  start_server them -public

  export DODDER_XDG_UTILITY_OVERRIDE="$us_home"
  export MADDER_XDG_UTILITY_OVERRIDE="$us_home"

  mkdir -p us
  pushd us || exit 1

  run_dodder_init

  run_dodder remote-add \
    toml-repo-uri-v0 \
    "http://${server_addr}" \
    them-http
  assert_success

  run_dodder pull /them-http +zettel,typ,etikett
  assert_success

  run_dodder show one/uno
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-gu738nunyrnsqukgqkuaau9zslu0fhwg4dgs9ltuyvnlp42wal8sdpn2hc !md "wow" tag]
	EOM
  popd || exit 1

  # them edits one/uno into a strict linear descendant.
  stop_server

  export DODDER_XDG_UTILITY_OVERRIDE="$them_home"
  export MADDER_XDG_UTILITY_OVERRIDE="$them_home"

  pushd them || exit 1
  edit_one_uno "wow the second" "edited remotely"
  popd || exit 1

  start_server them -public

  export DODDER_XDG_UTILITY_OVERRIDE="$us_home"
  export MADDER_XDG_UTILITY_OVERRIDE="$us_home"

  pushd us || exit 1

  run_dodder remote-add \
    toml-repo-uri-v0 \
    "http://${server_addr}" \
    them-http2
  assert_success

  # Second pull must fast-forward, not raise a spurious conflict.
  run_dodder pull /them-http2 +zettel,typ,etikett
  assert_success

  run_dodder show one/uno
  assert_success
  assert_output --regexp '^\[one/uno @blake2b256-.+ !md "wow the second" tag\]$'
  popd || exit 1
}

# pull_over_http_divergence_conflict: us and them both edit one/uno off a shared
# v0; us's pull must report a conflict, not silently accept them's version.
# Coverage for the already-implemented HTTP pull merge path.
function pull_over_http_divergence_conflict { # @test
  export DODDER_XDG_UTILITY_OVERRIDE="$them_home"
  export MADDER_XDG_UTILITY_OVERRIDE="$them_home"

  mkdir -p them
  (
    pushd them || exit 1
    run_dodder_init
    new_one_uno
  )

  start_server them -public

  export DODDER_XDG_UTILITY_OVERRIDE="$us_home"
  export MADDER_XDG_UTILITY_OVERRIDE="$us_home"

  mkdir -p us
  pushd us || exit 1

  run_dodder_init

  run_dodder remote-add \
    toml-repo-uri-v0 \
    "http://${server_addr}" \
    them-http
  assert_success

  run_dodder pull /them-http +zettel,typ,etikett
  assert_success

  # us diverges off v0.
  edit_one_uno "wow the local" "edited locally"
  popd || exit 1

  # them diverges independently off the same v0.
  stop_server

  export DODDER_XDG_UTILITY_OVERRIDE="$them_home"
  export MADDER_XDG_UTILITY_OVERRIDE="$them_home"

  pushd them || exit 1
  edit_one_uno "wow the remote" "edited remotely"
  popd || exit 1

  start_server them -public

  export DODDER_XDG_UTILITY_OVERRIDE="$us_home"
  export MADDER_XDG_UTILITY_OVERRIDE="$us_home"

  pushd us || exit 1

  run_dodder remote-add \
    toml-repo-uri-v0 \
    "http://${server_addr}" \
    them-http2
  assert_success

  # Genuine divergence: the pull must fail with a conflict, not overwrite.
  run_dodder pull /them-http2 +zettel,typ,etikett
  assert_failure
  assert_line --regexp 'conflicted.*\[one/uno\]'

  # The local divergent edit survives.
  run_dodder show one/uno
  assert_success
  assert_output --regexp '^\[one/uno @blake2b256-.+ !md "wow the local" tag\]$'
  popd || exit 1
}
