{
  inputs = {
    nixpkgs.url = "github:amarbel-llc/nixpkgs";
    nixpkgs-master.url = "github:NixOS/nixpkgs/e2dde111aea2c0699531dc616112a96cd55ab8b5";
    utils.url = "https://flakehub.com/f/numtide/flake-utils/0.1.102";

    bats = {
      url = "github:amarbel-llc/bats";
      inputs.nixpkgs.follows = "nixpkgs";
    };

    purse-first = {
      url = "github:amarbel-llc/purse-first";
      inputs.nixpkgs.follows = "nixpkgs";
      inputs.nixpkgs-master.follows = "nixpkgs-master";
      inputs.utils.follows = "utils";
    };

    bob = {
      url = "github:amarbel-llc/bob";
      inputs.nixpkgs.follows = "nixpkgs";
      inputs.nixpkgs-master.follows = "nixpkgs-master";
      inputs.utils.follows = "utils";
      inputs.purse-first.follows = "purse-first";
    };

    madder = {
      url = "github:amarbel-llc/madder";
      inputs.nixpkgs.follows = "nixpkgs";
      inputs.nixpkgs-master.follows = "nixpkgs-master";
      inputs.utils.follows = "utils";
      inputs.tommy.follows = "tommy";
      inputs.bob.follows = "bob";
    };

    tommy = {
      url = "github:amarbel-llc/tommy";
      inputs.utils.follows = "utils";
    };
  };

  outputs =
    {
      self,
      nixpkgs,
      utils,
      bats,
      bob,
      tommy,
      madder,
      ...
    }:
    let
      # Burnt into binaries via the fork's auto-injected -ldflags.
      # Single source of truth; `just bump-version` sed-rewrites this line.
      dodderVersion = "0.1.9";
      dodderCommit = self.shortRev or self.dirtyShortRev or "unknown";
    in
    (utils.lib.eachDefaultSystem (
      system:
      let
        result = import ./go/default.nix {
          inherit
            nixpkgs
            bats
            bob
            tommy
            madder
            system
            ;
          version = dodderVersion;
          commit = dodderCommit;
          man7Src = ./docs/man.7;
          batsSrc = ./zz-tests_bats;
        };
      in
      {
        inherit (result) packages;
        devShells.default = result.devShells.default;
      }
    ));
}
