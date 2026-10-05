# Nix side of go.mod for dodder. Carries both producer- and
# consumer-half of the flake-input-go_mod protocol (amarbel-llc/nixpkgs
# RFC 0001):
#
#   - producer: mkGoPkgs publishes go-pkgs / go-pkgs-test so future
#     downstream amarbel-llc consumers can bridge dodder's Go module
#     without organic gomod2nix.toml resolution (#217).
#
#   - consumer: goFlakeInputs routes the 8 cross-amarbel `require`
#     lines onto flake inputs, bypassing the organic gomod2nix.toml
#     hash and eliminating the flake.lock / go.mod / gomod2nix.toml
#     drift surface (#218). Consequence: for these 8 deps the go.mod
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
  piggy,
  tap,
  tommy,
  purse-first,
  chrest,
  system,
}:
rec {
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
      ".*\\.tmpl$"
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
    "code.linenisgreat.com/madder/go" = {
      src = madder.packages.${system}.go-pkgs;
    };
    # hyphence scopes its go-pkgs at /go (like madder), so no subPath.
    "code.linenisgreat.com/hyphence/go" = {
      src = hyphence.packages.${system}.go-pkgs;
    };
    # piggy owns the markl-id framework (piggy#183 inversion). Its
    # go-pkgs producer is scoped to go/ (module root, no subPath).
    "code.linenisgreat.com/piggy/go" = {
      src = piggy.packages.${system}.go-pkgs;
    };
    "code.linenisgreat.com/tap/go" = {
      src = tap.packages.${system}.go-pkgs;
      subPath = "go";
    };
    "code.linenisgreat.com/tommy" = {
      src = tommy.packages.${system}.go-pkgs;
    };
    "code.linenisgreat.com/purse-first/libs/dewey" = {
      src = purse-first.packages.${system}.go-pkgs;
      subPath = "libs/dewey";
    };
    "code.linenisgreat.com/purse-first/libs/go-mcp" = {
      src = purse-first.packages.${system}.go-pkgs;
      subPath = "libs/go-mcp";
    };
    # chrest builds with godyn (chrest#116): its go-pkgs is full-repo with
    # the rendered go.mod + gomod2nix.toml under go/, so slice with
    # subPath "go". Unlike the others, the go.mod require for chrest is
    # not a safe fallback — the published tree carries no committed
    # go.mod, so only this bridge resolves it (igloo FDR 0008).
    "code.linenisgreat.com/chrest/go" = {
      src = chrest.packages.${system}.go-pkgs;
      subPath = "go";
    };
  };

  # The bare-`go` escape hatches (`just go/test-go-pkg`,
  # `just generate-seed-types`) cannot resolve these modules through the
  # module proxy once a producer drops its committed go.mod (tommy, chrest:
  # igloo FDR 0008) — Go then compiles the fetched tree as go1.16. This file
  # holds one go.work `replace` line per bridged module, pointing at the same
  # go-pkgs store path the nix builds compile, so a throwaway go.work built
  # from it gives bare `go` the identical sources. Consumed by the
  # `_go-work-flake-inputs` recipe in go/justfile.
  goWorkReplaces = pkgs.writeText "dodder-go-work-replaces" (
    pkgs.lib.concatStrings (
      pkgs.lib.mapAttrsToList (
        module: input:
        "replace ${module} => ${input.src}${
          pkgs.lib.optionalString (input ? subPath) "/${input.subPath}"
        }\n"
      ) goFlakeInputs
    )
  );
}
