{
  nixpkgs,
  nixpkgs-master,
  purse-first ? null,
  bob ? null,
  just-us,
  tommy,
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

  devShells.default = pkgs-master.mkShell {
    packages = [
      just-us.packages.${system}.just
      tommy.packages.${system}.default
    ]
    ++ (with pkgs-master; [
      fish
      gnumake
      gum
      httpie
      pandoc
    ])
    ++ pkgs-master.lib.optionals (bob != null) [
      bob.packages.${system}.batman
      bob.packages.${system}.tap-dancer
    ];

    inputsFrom = pkgs-master.lib.optionals (purse-first != null) [
      purse-first.devShells.${system}.go
      purse-first.devShells.${system}.shell
    ];
  };
}
