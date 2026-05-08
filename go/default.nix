{
  nixpkgs,
  bob ? null,
  tommy,
  madder ? null,
  system,
  man7Src ? null,
  # Passed to buildGoApplication's `version` and `commit` attrs; the
  # fork's nixpkgs auto-injects them as `-X main.version` and
  # `-X main.commit` ldflags on every subPackage. Defaulted so direct
  # `import ./go/default.nix` (outside the flake) still works; release
  # builds always override via flake.nix.
  version ? "dev",
  commit ? "unknown",
}:
let
  pkgs = import nixpkgs {
    inherit system;
  };

  dodder = pkgs.buildGoApplication {
    pname = "dodder";
    inherit version commit;
    src = ./.;
    pwd = ./.;
    subPackages = [
      "cmd/der"
      "cmd/dodder"
      "cmd/dodder-gen_man"
    ];
    modules = ./gomod2nix.toml;
    go = pkgs.go_1_26;
    GOTOOLCHAIN = "local";

    nativeBuildInputs = pkgs.lib.optionals (man7Src != null) [
      pkgs.pandoc
    ];

    postInstall = ''
      mkdir -p $out/share/man/man1
      $out/bin/dodder-gen_man $out/share/man/man1
      rm $out/bin/dodder-gen_man
    ''
    + pkgs.lib.optionalString (man7Src != null) ''
      mkdir -p $out/share/man/man7
      for f in ${man7Src}/*.md; do
        name="$(basename "$f" .md)"
        pandoc -s -t man "$f" -o "$out/share/man/man7/$name.7"
        # .ss 12 0 = disable double sentence spacing
        # .na = ragged-right (no justification)
        ${pkgs.gnused}/bin/sed -i '3a\.\\" Formatting overrides\n.ss 12 0\n.na' "$out/share/man/man7/$name.7"
      done
    '';
  };
in
{
  packages = {
    inherit dodder;
    default = dodder;
  };

  docker = pkgs.dockerTools.buildImage {
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

  devShells.default = pkgs.mkShell {
    packages = [
      pkgs.gomod2nix
      tommy.packages.${system}.default
    ]
    ++ (with pkgs; [
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
      just
      lsof
      pandoc
      radicale
      shellcheck
      shfmt
      yq-go
    ])
    ++ pkgs.lib.optionals (bob != null) [
      bob.packages.${system}.batman
      bob.packages.${system}.tap-dancer
    ]
    ++ pkgs.lib.optionals (madder != null) [
      madder.packages.${system}.default
    ];

    GOTOOLCHAIN = "local";
  };
}
