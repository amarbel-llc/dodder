{
  nixpkgs,
  nixpkgs-master,
  purse-first,
  gomod2nix,
  system,
}:
let
  pkgs = import nixpkgs {
    inherit system;
    overlays = [
      gomod2nix.overlays.default
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
      "cmd/dodder"
      "cmd/mad"
      "cmd/madder"
    ];
    modules = ./gomod2nix.toml;
    go = pkgs.go_1_25;
    GOTOOLCHAIN = "local";
  };
in
{
  packages = {
    inherit dodder;
    default = dodder;
  };

  docker = pkgs-master.dockerTools.buildImage {
    name = "dodder";
    tag = "latest";
    copyToRoot = [ dodder ];
    config = {
      Cmd = [ "${dodder}/bin/dodder" ];
      Env = [ ];
      ExposedPorts = {
        "9000/tcp" = { };
      };
    };
  };

  devShell = pkgs-master.mkShell {
    packages =
      (with pkgs-master; [
        fish
        gnumake
        gum
      ])
      ++ [
        purse-first.packages.${system}.batman
        purse-first.packages.${system}.tap-dancer
      ];

    inputsFrom = [
      purse-first.devShells.${system}.go
      purse-first.devShells.${system}.shell
    ];
  };
}
