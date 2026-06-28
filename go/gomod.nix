# Nix side of go.mod for dodder. Carries both producer- and
# consumer-half of the flake-input-go_mod protocol (amarbel-llc/nixpkgs
# RFC 0001):
#
#   - producer: mkGoPkgs publishes go-pkgs / go-pkgs-test so future
#     downstream amarbel-llc consumers can bridge dodder's Go module
#     without organic gomod2nix.toml resolution (#217).
#
#   - consumer: goFlakeInputs routes the 5 cross-amarbel `require`
#     lines onto flake inputs, bypassing the organic gomod2nix.toml
#     hash and eliminating the flake.lock / go.mod / gomod2nix.toml
#     drift surface (#218). Consequence: for these 5 deps the go.mod
#     rev (and the gomod2nix.toml hash) is VESTIGIAL — every
#     buildGoApplication here (release, dodder-debug, dodder-go-test,
#     race, cover, bats lanes) inherits goFlakeInputs and so compiles
#     them from the flake-input source, never from go.mod. flake.lock
#     is the source of truth. The go.mod require line stays for Go
#     module validity and the bare-`go test` / devshell escape hatches;
#     update-flake-input / resync-flake-go keep its shadow rev aligned.
#
# Mixed-flake shape per RFC 0001 § The `gomod.nix` convention. Single
# place to add/remove either side; flake.nix imports once and passes
# the relevant outputs into go/default.nix.
#
# Keep all gomod2nix.toml consumers in sync: a buildGoApplication
# call that forgets `goFlakeInputs` sees the unmerged module graph
# and resurrects the lockstep.
{
  pkgs,
  src,
  madder,
  hyphence,
  tap,
  tommy,
  purse-first,
  system,
}:
{
  # mkGoPkgs defaults drop non-Go files; assets `//go:embed`ed at
  # compile time would otherwise vanish from the filtered source tree:
  # the pandoc filters/defaults under
  # internal/romeo/local_working_copy/embedded/ (.lua/.yaml) and the
  # default zettel-id word lists under
  # internal/echo/zettel_id_provider/embedded/ (.txt). Extras patterns
  # are anchored regexps against the repo-relative path; the .txt rule
  # is scoped to embedded/ dirs so unrelated .txt files stay dropped.
  #
  # TODO[amarbel-llc/nixpkgs#60]: mkGoPkgs could derive these extras
  # automatically from `//go:embed` directives so consumers don't have
  # to hand-maintain the list.
  goPkgs = pkgs.mkGoPkgs {
    inherit src;
    extras = [
      ".*\\.lua$"
      ".*\\.yaml$"
      ".*/embedded/.*\\.txt$"
    ];
  };

  # Bridging cross-amarbel deps through their own `go-pkgs` outputs
  # means non-Go edits in those repos no longer trigger dodder
  # rebuilds, and bumping their flake-input rev is enough to pick up
  # new code (no go.mod / gomod2nix.toml edit required for the Nix
  # build path).
  #
  # subPath semantics depend on the producer's own scoping choice:
  #   - madder scopes its go-pkgs at /go (madder/flake.nix passes
  #     `src = self + "/go"` to mkGoPkgs), so consumers omit subPath.
  #   - tap publishes a full-repo go-pkgs, so consumers slice with
  #     `subPath = "go"`.
  #   - tommy's module sits at its repo root.
  #   - purse-first publishes a full-repo go-pkgs, so consumers slice
  #     into the relevant library subdirectory.
  goFlakeInputs = {
    "github.com/amarbel-llc/madder/go" = {
      src = madder.packages.${system}.go-pkgs;
    };
    # hyphence scopes its go-pkgs producer at /go (like madder), so the
    # module root maps directly with no subPath. madder#253 extracted
    # the hyphence format library here; dodder#295 consumes it.
    "github.com/amarbel-llc/hyphence/go" = {
      src = hyphence.packages.${system}.go-pkgs;
    };
    "github.com/amarbel-llc/tap/go" = {
      src = tap.packages.${system}.go-pkgs;
      subPath = "go";
    };
    "github.com/amarbel-llc/tommy" = {
      src = tommy.packages.${system}.go-pkgs;
    };
    "github.com/amarbel-llc/purse-first/libs/dewey" = {
      src = purse-first.packages.${system}.go-pkgs;
      subPath = "libs/dewey";
    };
    "github.com/amarbel-llc/purse-first/libs/go-mcp" = {
      src = purse-first.packages.${system}.go-pkgs;
      subPath = "libs/go-mcp";
    };
  };
}
