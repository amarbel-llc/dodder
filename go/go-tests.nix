# Sandboxed Go unit test lane for dodder.
#
# Runs `go test -tags test,debug ./...` inside a nix-build sandbox so the
# tests cannot leak into the developer's real $HOME / XDG dirs. Modeled on
# go/bats.nix's bats lane but for Go unit tests instead of bats integration
# tests.
#
# Why: tests that exercise the env_repo / blob_store machinery transitively
# call into madder's MakeBlobStores, which intentionally merges the real
# user's XDG blob stores into any process-level override (see
# madder/internal/foxtrot/blob_stores/main.go). The only reliable
# isolation is OS-level: HOME=/build, XDG_*_HOME under /build, no network.
{
  pkgs,
  version,
  commit,
}:
let
  dodder-go-test = pkgs.buildGoApplication {
    pname = "dodder-go-test";
    inherit version commit;
    src = ./.;
    pwd = ./.;
    # buildGoApplication insists on at least one subPackage to build. The
    # produced binary is incidental — only the checkPhase matters here.
    subPackages = [ "cmd/dodder" ];
    modules = ./gomod2nix.toml;
    go = pkgs.go_1_26;
    GOTOOLCHAIN = "local";
    # NOTE: this fork's goBuildHook splats `tags` as separate argv entries
    # after `-tags`, so multiple elements break with `package X is not in std`.
    # Use a single comma-joined element instead — `go build -tags test,debug`
    # parses the value identically.
    tags = [ "test,debug" ];

    doCheck = true;
    preCheck = ''
      # The sandbox roots at /build. Carve out per-utility XDG dirs underneath
      # so anything that walks $HOME or $XDG_*_HOME lands inside the sandbox
      # rather than the developer's real ~/.local/share/madder/ etc.
      export HOME=$TMPDIR/home
      export XDG_DATA_HOME=$TMPDIR/xdg-data
      export XDG_CONFIG_HOME=$TMPDIR/xdg-config
      mkdir -p $HOME $XDG_DATA_HOME $XDG_CONFIG_HOME

      # Mirror go/bats.nix:140-144 — ceiling the workspace walk at /build
      # prevents env_workspace from punching out via findWorkspaceFile.
      export DODDER_CEILING_DIRECTORIES=/build
      export MADDER_CEILING_DIRECTORIES=/build

      # Mirror go/bats.nix:136 — match the default the bats lane uses so we
      # behave the same under memory pressure.
      export GOMEMLIMIT=512MiB
    '';
    # Run the full test tree, not just subPackages.
    checkPhase = ''
      runHook preCheck
      go test -count=1 -tags test,debug ./...
      runHook postCheck
    '';
    # We don't need the placeholder binary. Replace the standard install
    # phase with a marker file so $out is non-empty (nix requires it).
    installPhase = ''
      mkdir -p $out
      touch $out/.tested
    '';
  };
in
{
  inherit dodder-go-test;
}
