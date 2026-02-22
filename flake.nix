{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/6d41bc27aaf7b6a3ba6b169db3bd5d6159cfaa47";
    utils.url = "https://flakehub.com/f/numtide/flake-utils/0.1.102";

    dodder-go.url = "path:./go";
  };

  outputs =
    {
      self,
      utils,
      dodder-go, nixpkgs,
    }:
    (utils.lib.eachDefaultSystem (
      system:
      {
        packages = dodder-go.packages.${system};
        devShells = dodder-go.devShells.${system};
      }
    ));
}
