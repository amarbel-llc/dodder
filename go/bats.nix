# bats integration test lanes for dodder.
#
# Wraps the `batsLane` function from amarbel-llc/bats's
# `lib.${system}.batsLane` (same source as the nixpkgs overlay's
# `pkgs.testers.batsLane`, but tracked in the bats repo so the
# dependency lives alongside `bats-libs` rather than spread across
# two flake inputs).
#
# Dodder-specific defaults: `bats-libs` from amarbel-llc/bats on
# `BATS_LIB_PATH`, DODDER_BIN / DODDER_DER_BIN /
# DODDER_TEST_SFTP_SERVER / MADDER_BIN exported via the `binaries`
# map, `BATS_TEST_TIMEOUT` mirroring `zz-tests_bats/justfile`. The
# `testFiles` parameter (amarbel-llc/nixpkgs#24, mirrored in
# amarbel-llc/bats) explicitly enumerates dodder's two-subdir layout
# so the lane builder doesn't fall back to its default top-level
# `*.bats` glob, which doesn't match anything in dodder's tree.
#
# Auto-discovers `# bats file_tags=foo,bar` directives at flake-eval
# time across the runnable test files, producing one `bats-${tag}`
# derivation per unique tag plus `bats-default` (no filter). The
# frozen snapshot dirs `previous_versions/v*/` are explicitly skipped
# — their .bats files are historical reference (sourced by the
# snapshot system, not run as live tests).
{
  pkgs,
  # batsLane: the lane-builder function from
  # `bats.lib.${system}.batsLane` (amarbel-llc/bats). Threaded in
  # rather than re-imported here so the caller controls the bats
  # flake-input pin.
  batsLane,
  bats-libs,
  dodder,
  dodder-test-sftp-server,
  madder-bin,
  batsSrc,
  # Source-of-truth flake.nix, staged at stage/flake.nix so the
  # version-burnin test (current_version/version.bats) can `grep
  # dodderVersion = ` against it via ${BATS_TEST_DIRNAME}/../../flake.nix.
  flakeNixSrc,
  # Worktree-root zz-pandoc-refs/ directory holding pandoc filter
  # scripts (currently just discover-refs.lua). Staged at
  # stage/zz-pandoc-refs/ so show_zettel_with_pandoc_discovered_*
  # can resolve the script via ${BATS_TEST_DIRNAME}/../../zz-pandoc-refs/.
  # Requires the extraStagedFiles mkdir -p fix from amarbel-llc/bats#10.
  pandocRefsSrc,
  batsTestTimeout ? "30",
}:
let
  inherit (pkgs) lib;

  # Runnable test set. previous_versions/v*/*.bats are NOT included
  # here — they are frozen snapshots staged for migration tests to
  # consult, not invoked directly.
  testFiles = [
    "current_version/*.bats"
    "previous_versions/main.bats"
  ];

  mkDodderBatsLane =
    {
      filter ? "",
      base ? dodder,
    }:
    batsLane {
      inherit base filter batsSrc testFiles;
      binaries = {
        DODDER_BIN = {
          inherit base;
          name = "dodder";
        };
        DODDER_DER_BIN = {
          inherit base;
          name = "der";
        };
        DODDER_TEST_SFTP_SERVER = {
          base = dodder-test-sftp-server;
          name = "dodder-test-sftp-server";
        };
        MADDER_BIN = {
          base = madder-bin;
          name = "madder";
        };
      };
      batsLibPath = [ bats-libs.batsLibPath ];
      # The lane runs in a stripped nix sandbox — anything not listed
      # here is genuinely absent, including binaries the tests call
      # via bare PATH lookups (rather than through DODDER_BIN /
      # MADDER_BIN env vars). Each entry is load-bearing for at least
      # one test file; the comment names the test or test family.
      #
      # tree:    migration_reindex prints `tree .madder` for diagnostic
      #          output (previous_versions/main.bats:41).
      # curl:    serve.bats probes /healthz and other HTTP endpoints
      #          after `dodder serve` binds 127.0.0.1 (current_version/serve.bats).
      # openssh: ssh-agent + ssh-keygen + ssh-add for the af_unix lane
      #          (current_version/init_ecdsa_p256.bats).
      # git:     the pull/push lanes shell out to git as part of the
      #          remote-transfer workflow (current_version/pull.bats,
      #          push.bats, mergetool.bats).
      # pandoc:  invoked by dodder when a zettel/type uses a !ref-pandoc-*
      #          script handler (current_version/show.bats,
      #          format_pandoc.bats, several fields.bats cases).
      # vim:     mergetool tests resolve the `vimdiff` script
      #          (current_version/mergetool.bats).
      # dodder + madder + dodder-test-sftp-server: a handful of tests
      #          invoke these binaries by bare name on PATH rather than
      #          through their DODDER_BIN / MADDER_BIN /
      #          DODDER_TEST_SFTP_SERVER env-var aliases. Putting the
      #          packages on nativeBuildInputs gives both surfaces.
      nativeBuildInputs = (with pkgs; [
        tree
        curl
        openssh
        git
        pandoc
        vim
        # yq-go: the field-script tests pipe blob contents through
        #        `yq -p toml -o json '...'` to project structured
        #        fields (current_version/fields.bats family).
        yq-go
      ]) ++ [
        dodder
        dodder-test-sftp-server
        madder-bin
      ];
      extraStagedFiles = [
        { src = flakeNixSrc; dest = "flake.nix"; }
        # show_zettel_with_pandoc_discovered_* call pandoc with a Lua
        # filter that lives in zz-pandoc-refs/ at the worktree root,
        # outside zz-tests_bats/. Stage it parallel to the bats source
        # so the tests' relative path resolves the same as in dev.
        { src = pandocRefsSrc + "/discover-refs.lua"; dest = "zz-pandoc-refs/discover-refs.lua"; }
      ];
      extraEnv = {
        BATS_TEST_TIMEOUT = batsTestTimeout;
        GOMEMLIMIT = "512MiB";
        # Empty matches the existing zz-tests_bats/justfile default —
        # tests that need a specific override set it themselves.
        DODDER_XDG_UTILITY_OVERRIDE = "";
        # The nix sandbox roots at /build; ceiling the workspace walk
        # there prevents dodder from punching out of the sandbox via
        # findWorkspaceFile.
        DODDER_CEILING_DIRECTORIES = "/build";
        MADDER_CEILING_DIRECTORIES = "/build";
        # Tests that exercise `dodder edit` / `dodder new -edit=true`
        # spawn $EDITOR; in the sandbox there's no interactive editor,
        # so route them through `true` (no-op, exit 0) — the surrounding
        # tests assert outcomes other than the editor's behavior.
        EDITOR = "true";
      };
    };

  # Recursive walker over batsSrc. Skips previous_versions/v*/
  # snapshot dirs (and only those — a future top-level dir starting
  # with 'v' would not be excluded).
  walkBats =
    dir: prefix:
    let
      entries = builtins.readDir dir;
      isSnapshotDir =
        name: type:
        type == "directory"
        && prefix == "previous_versions/"
        && (builtins.match "v[0-9]+" name) != null;
    in
    lib.flatten (
      lib.mapAttrsToList (
        name: type:
        if isSnapshotDir name type then
          [ ]
        else if type == "directory" then
          walkBats (dir + "/${name}") "${prefix}${name}/"
        else if lib.hasSuffix ".bats" name then
          [ "${prefix}${name}" ]
        else
          [ ]
      ) entries
    );

  batsFiles = walkBats batsSrc "";

  # Trim leading/trailing whitespace from each tag — `# bats
  # file_tags=foo, bar` is common and the leading space on `bar`
  # would otherwise produce a derivation named `bats- bar` which
  # nix rejects.
  trim = s: lib.removePrefix " " (lib.removeSuffix " " s);

  extractFileTags =
    file:
    let
      content = builtins.readFile (batsSrc + "/${file}");
      lines = lib.splitString "\n" content;
      tagLines = lib.filter (l: lib.hasPrefix "# bats file_tags=" l) lines;
    in
    if tagLines == [ ] then
      [ ]
    else
      map trim (
        lib.splitString "," (lib.removePrefix "# bats file_tags=" (builtins.head tagLines))
      );

  allFileTags = lib.unique (lib.concatMap extractFileTags batsFiles);

  batsLaneOutputs =
    lib.listToAttrs (
      map (tag: lib.nameValuePair "bats-${tag}" (mkDodderBatsLane { filter = tag; })) allFileTags
    )
    // {
      bats-default = mkDodderBatsLane { };
    };

  # Hermetic fixture generator. Runs the bats fixture script
  # (previous_versions/generate_fixture.bats) inside the sandbox,
  # captures the surviving BATS_RUN_TMPDIR, copies .dodder + .madder
  # into $out/v${storeVersion}/, and computes .fixtures.env values
  # by re-running dodder against the captured store. Output is meant
  # to be materialized into the worktree by a justfile recipe — this
  # derivation itself does not write to the worktree.
  fixtures-current = pkgs.runCommand "dodder-fixtures" {
    nativeBuildInputs = [
      pkgs.bats
      pkgs.parallel
      dodder
    ];
  } ''
    set -euo pipefail

    export HOME=$TMPDIR
    export DODDER_BIN=${dodder}/bin/dodder
    export DODDER_DER_BIN=${dodder}/bin/der
    export DODDER_TEST_SFTP_SERVER=${dodder-test-sftp-server}/bin/dodder-test-sftp-server
    export MADDER_BIN=${madder-bin}/bin/madder
    export DODDER_VERSION="v$(${dodder}/bin/dodder info store-version)"
    export DODDER_CEILING_DIRECTORIES=/build
    export MADDER_CEILING_DIRECTORIES=/build
    export DODDER_XDG_UTILITY_OVERRIDE=
    export GOMEMLIMIT=512MiB
    export BATS_LIB_PATH="${bats-libs.batsLibPath}"
    export BATS_TEST_TIMEOUT=30

    # Stage a writable copy of the bats source; bats walks it from
    # CWD and writes to BATS_RUN_TMPDIR underneath.
    mkdir -p stage
    cp -r ${batsSrc}/. stage/
    chmod -R u+w stage
    cd stage

    # --no-tempdir-cleanup keeps the per-test scratch alive so we
    # can lift .dodder/.madder out of it. The 2>&1 is load-bearing:
    # bats prints the `BATS_RUN_TMPDIR: ...` summary line to STDERR,
    # and we grep for it below. (Mirrors the existing zz-tests_bats/
    # previous_versions/generate_fixture.bash:17 — tee without 2>&1
    # silently drops the line and the script exits before the missing-
    # dir error message can fire.)
    bats --no-tempdir-cleanup \
         previous_versions/generate_fixture.bats 2>&1 \
      | tee bats.out

    bats_dir=$(grep "BATS_RUN_TMPDIR" bats.out | head -1 | cut -d' ' -f2)
    if [[ -z "$bats_dir" ]]; then
      echo "ERROR: BATS_RUN_TMPDIR not found in bats output" >&2
      exit 1
    fi
    if [[ ! -d "$bats_dir/test/1/.dodder" ]]; then
      echo "ERROR: expected fixture dir not found: $bats_dir/test/1/.dodder" >&2
      exit 1
    fi

    out_dir="$out/$DODDER_VERSION"
    mkdir -p "$out_dir"
    cp -r "$bats_dir/test/1/.dodder" "$out_dir/.dodder"
    cp -r "$bats_dir/test/1/.madder" "$out_dir/.madder"
    chmod -R u+w "$out_dir"

    # Extract fixture-specific values for test assertions. Mirrors
    # zz-tests_bats/previous_versions/generate_fixture.bash verbatim
    # — sigs and shas are load-bearing.
    pushd "$out_dir" >/dev/null

    type_sig=$(${dodder}/bin/dodder show -format type-sig one/uno)
    [[ -n "$type_sig" ]] || { echo "ERROR: type_sig extraction failed" >&2; exit 1; }

    konfig_sha=$(${dodder}/bin/dodder show \
      -abbreviate-shas=false \
      -print-empty-shas=true \
      -format log :konfig | sed 's/.*@\([^ ]*\) .*/\1/')
    [[ -n "$konfig_sha" ]] || { echo "ERROR: konfig_sha extraction failed" >&2; exit 1; }

    type_blob_sha=$(${dodder}/bin/dodder show \
      -abbreviate-shas=false \
      -print-empty-shas=true \
      -format log '!md:t' | sed 's/.*@\([^ ]*\) .*/\1/')
    [[ -n "$type_blob_sha" ]] || { echo "ERROR: type_blob_sha extraction failed" >&2; exit 1; }

    cat > .fixtures.env <<EOF
# Auto-generated by the fixtures-current nix derivation -- do not edit
FIXTURE_TYPE_SIG=$type_sig
FIXTURE_KONFIG_SHA=$konfig_sha
FIXTURE_TYPE_BLOB_SHA=$type_blob_sha
EOF

    # Sanity checks: the generated store actually contains the
    # zettel + konfig the suite expects.
    ${dodder}/bin/dodder show one/uno >/dev/null \
      || { echo "ERROR: fixture store missing zettel one/uno" >&2; exit 1; }
    ${dodder}/bin/dodder show :konfig >/dev/null \
      || { echo "ERROR: fixture store missing konfig" >&2; exit 1; }

    popd >/dev/null
  '';
in
{
  inherit mkDodderBatsLane batsLaneOutputs fixtures-current;
}
