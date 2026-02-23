{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/23d72dabcb3b12469f57b37170fcbc1789bd7457";
    nixpkgs-master.url = "github:NixOS/nixpkgs/b28c4999ed71543e71552ccfd0d7e68c581ba7e9";
    utils.url = "https://flakehub.com/f/numtide/flake-utils/0.1.102";

    devenv-go = {
      url = "github:amarbel-llc/purse-first?dir=devenvs/go";
      inputs.nixpkgs.follows = "nixpkgs";
      inputs.nixpkgs-master.follows = "nixpkgs-master";
      inputs.utils.follows = "utils";
    };

    devenv-shell = {
      url = "github:amarbel-llc/purse-first?dir=devenvs/shell";
      inputs.nixpkgs.follows = "nixpkgs";
      inputs.nixpkgs-master.follows = "nixpkgs-master";
      inputs.utils.follows = "utils";
    };

    purse-first = {
      url = "github:amarbel-llc/purse-first";
      inputs.nixpkgs.follows = "nixpkgs";
      inputs.nixpkgs-master.follows = "nixpkgs-master";
      inputs.utils.follows = "utils";
    };
  };

  outputs =
    { self
    , nixpkgs
    , nixpkgs-master
    , utils
    , devenv-go
    , devenv-shell
    , purse-first
    ,
    }:
    (utils.lib.eachDefaultSystem
      (system:
      let

        pkgs = import nixpkgs {
          inherit system;

          overlays = [
            devenv-go.overlays.default
          ];
        };

        pkgs-master = import nixpkgs-master {
          inherit system;
        };

        dodder = pkgs.buildGoApplication {
          pname = "dodder";
          version = "0.0.1";
          src = ./.;
          subPackages = [
            "cmd/der"
            # "cmd/dodder"
            "cmd/mad"
            "cmd/madder"
          ];
          modules = ./gomod2nix.toml;
          go = pkgs.go_1_25;
          GOTOOLCHAIN = "local";
        };

      in
      {

        packages.dodder = dodder;
        packages.default = dodder;

        docker = pkgs-master.dockerTools.buildImage {
          name = "dodder";
          tag = "latest";
          copyToRoot = [ dodder ];
          config = {
            Cmd = [ "${dodder}/bin/dodder" ];
            Env = [ ];
            ExposedPorts = { "9000/tcp" = { }; };
          };
        };

        devShells.default = pkgs-master.mkShell {
          # inherit (gomod2nix.packages.${system}) mkGoEnv gomod2nix;

          packages = (with pkgs-master; [
            fish
            gnumake
            gum
          ]) ++ [
            purse-first.packages.${system}.batman
            purse-first.packages.${system}.tap-dancer
          ];

          inputsFrom = [
            devenv-go.devShells.${system}.default
            devenv-shell.devShells.${system}.default
          ];
        };
      })
    );
}
