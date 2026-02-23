{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/23d72dabcb3b12469f57b37170fcbc1789bd7457";
    nixpkgs-master.url = "github:NixOS/nixpkgs/b28c4999ed71543e71552ccfd0d7e68c581ba7e9";
    utils.url = "https://flakehub.com/f/numtide/flake-utils/0.1.102";

    purse-first = {
      url = "github:amarbel-llc/purse-first";
      inputs.nixpkgs.follows = "nixpkgs";
      inputs.nixpkgs-master.follows = "nixpkgs-master";
      inputs.utils.follows = "utils";
    };
  };

  outputs =
    {
      self,
      nixpkgs,
      nixpkgs-master,
      utils,
      purse-first,
    }:
    (utils.lib.eachDefaultSystem (
      system:
      let
        result = import ./go/default.nix {
          inherit
            nixpkgs
            nixpkgs-master
            purse-first
            system
            ;
        };
      in
      {
        inherit (result) packages;
        devShells.default = result.devShell;
      }
    ));
}
