{
  description = "";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/23d72dabcb3b12469f57b37170fcbc1789bd7457";
    nixpkgs-master.url = "github:NixOS/nixpkgs/b28c4999ed71543e71552ccfd0d7e68c581ba7e9";
    utils.url = "https://flakehub.com/f/numtide/flake-utils/0.1.102";
  };

  outputs =
    {
      self,
      nixpkgs,
      nixpkgs-master,
      utils,
    }:
    utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs { inherit system; };
        name = "generate";
        buildInputs = with pkgs; [
          jq
          yq-go
          just
        ];
        generate = (pkgs.writeScriptBin name (builtins.readFile ./generate.bash)).overrideAttrs (old: {
          buildCommand = "${old.buildCommand}\n patchShebangs $out";
        });
      in
      rec {
        packages.generate = pkgs.symlinkJoin {
          name = name;
          paths = [ generate ] ++ buildInputs;
          buildInputs = [ pkgs.makeWrapper ];
          postBuild = "wrapProgram $out/bin/${name} --prefix PATH : $out/bin";
        };

        defaultPackage = packages.generate;

        devShells.default = pkgs.mkShell {
          packages = (
            with pkgs;
            [
              jq
              yq-go
              just
            ]
          );

          inputsFrom = [ ];
        };
      }
    );
}
