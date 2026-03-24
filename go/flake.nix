{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/23d72dabcb3b12469f57b37170fcbc1789bd7457";
    nixpkgs-master.url = "github:NixOS/nixpkgs/b28c4999ed71543e71552ccfd0d7e68c581ba7e9";
    utils.url = "https://flakehub.com/f/numtide/flake-utils/0.1.102";

    gomod2nix = {
      url = "github:nix-community/gomod2nix";
      inputs.nixpkgs.follows = "nixpkgs-master";
    };

    purse-first = {
      url = "github:amarbel-llc/purse-first";
      inputs.nixpkgs.follows = "nixpkgs";
      inputs.nixpkgs-master.follows = "nixpkgs-master";
      inputs.utils.follows = "utils";
    };

    bob = {
      url = "github:amarbel-llc/bob";
      inputs.purse-first.follows = "purse-first";
      inputs.nixpkgs.follows = "nixpkgs";
      inputs.nixpkgs-master.follows = "nixpkgs-master";
      inputs.utils.follows = "utils";
    };

    just-us = {
      url = "github:amarbel-llc/just-us";
      inputs.nixpkgs.follows = "nixpkgs";
      inputs.nixpkgs-master.follows = "nixpkgs-master";
      inputs.utils.follows = "utils";
      inputs.purse-first.follows = "purse-first";
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
      nixpkgs-master,
      utils,
      gomod2nix,
      purse-first,
      bob,
      just-us,
      tommy,
    }:
    (utils.lib.eachDefaultSystem (
      system:
      let
        result = import ./default.nix {
          inherit
            nixpkgs
            nixpkgs-master
            gomod2nix
            bob
            just-us
            tommy
            system
            ;
        };
      in
      {
        inherit (result) packages docker;
        devShells.default = result.devShells.default;
      }
    ));
}
