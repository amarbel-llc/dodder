{
  nixpkgs,
  nixpkgs-master,
  bob ? null,
  tommy,
  gomod2nix,
  system,
  man7Src ? null,
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
      "cmd/dodder-gen_man"
      "cmd/mad"
      "cmd/madder"
    ];
    modules = ./gomod2nix.toml;
    go = pkgs-master.go_1_26;
    GOTOOLCHAIN = "local";

    nativeBuildInputs = pkgs-master.lib.optionals (man7Src != null) [
      pkgs-master.pandoc
    ];

    postInstall = ''
      mkdir -p $out/share/man/man1
      $out/bin/dodder-gen_man $out/share/man/man1
      rm $out/bin/dodder-gen_man
    ''
    + pkgs-master.lib.optionalString (man7Src != null) ''
      mkdir -p $out/share/man/man7
      for f in ${man7Src}/*.md; do
        name="$(basename "$f" .md)"
        pandoc -s -t man "$f" -o "$out/share/man/man7/$name.7"
        # .ss 12 0 = disable double sentence spacing
        # .na = ragged-right (no justification)
        ${pkgs-master.gnused}/bin/sed -i '3a\.\\" Formatting overrides\n.ss 12 0\n.na' "$out/share/man/man7/$name.7"
      done
    '';
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
      yq-go
    ])
    ++ pkgs-master.lib.optionals (bob != null) [
      bob.packages.${system}.batman
      bob.packages.${system}.tap-dancer
    ];

    GOTOOLCHAIN = "local";
  };
}
