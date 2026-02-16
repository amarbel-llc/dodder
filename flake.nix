{
  inputs = {
    utils.url = "https://flakehub.com/f/numtide/flake-utils/0.1.102";

    dodder-go.url = "path:./go";
  };

  outputs =
    {
      self,
      utils,
      dodder-go,
    }:
    (utils.lib.eachDefaultSystem (
      system:
      {
        packages = dodder-go.packages.${system};
        devShells = dodder-go.devShells.${system};
      }
    ));
}
