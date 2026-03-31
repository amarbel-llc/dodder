{
  nixpkgs,
  nixpkgs-master,
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
    pwd = ./.;
    subPackages = [
      "cmd/der"
      "cmd/dodder"
      "cmd/mad"
      "cmd/madder"
    ];
    modules = ./gomod2nix.toml;
    go = pkgs-master.go_1_26;
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
      gomod2nix.packages.${system}.default
      just-us.packages.${system}.just
      tommy.packages.${system}.default
    ]
    ++ (with pkgs-master; [
      bash-language-server
      delve
      fish
      gnumake
      go_1_26
      gofumpt
      golangci-lint
      golines
      gopls
      gotools
      govulncheck
      gum
      httpie
      lsof
      pandoc
      radicale
      shellcheck
      shfmt
    ])
    ++ pkgs-master.lib.optionals (bob != null) [
      bob.packages.${system}.batman
      bob.packages.${system}.tap-dancer
    ];

    GOTOOLCHAIN = "local";
  };
}
