{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/23d72dabcb3b12469f57b37170fcbc1789bd7457";
    nixpkgs-master.url = "github:NixOS/nixpkgs/b28c4999ed71543e71552ccfd0d7e68c581ba7e9";
    utils.url = "https://flakehub.com/f/numtide/flake-utils/0.1.102";

    devenv-go.url = "github:friedenberg/eng?dir=devenvs/go";
    devenv-shell.url = "github:friedenberg/eng?dir=devenvs/shell";
    devenv-bats.url = "github:friedenberg/eng?dir=devenvs/bats";
    sandcastle.url = "github:amarbel-llc/sandcastle";
    bats-assert-additions.url = "github:amarbel-llc/bats-assert-additions";
  };

  outputs =
    { self
    , nixpkgs
    , nixpkgs-master
    , utils
    , devenv-go
    , devenv-shell
    , devenv-bats
    , sandcastle
    , bats-assert-additions
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
            bats
            bats.libraries.bats-support
            bats.libraries.bats-assert
            fish
            gnumake
            gum
          ]) ++ [
            sandcastle.packages.${system}.default
            bats-assert-additions.packages.${system}.default
          ];

          inputsFrom = [
            devenv-go.devShells.${system}.default
            devenv-shell.devShells.${system}.default
            devenv-bats.devShells.${system}.default
          ];

          BATS_LIB_PATH = "${pkgs-master.bats.libraries.bats-support}/share/bats:${pkgs-master.bats.libraries.bats-assert}/share/bats:${bats-assert-additions.packages.${system}.default}/share/bats";
        };
      })
    );
}
