{
  description = "gitutils";

  inputs.nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
  inputs.flake-utils.url = "github:numtide/flake-utils";

  outputs = {
    self,
      nixpkgs,
      flake-utils,
  }: (flake-utils.lib.eachDefaultSystem (system:
    let
      pkgs = import nixpkgs {
        inherit system;
        config.allowUnfree = true;
      };
      callPackage = pkgs.lib.callPackageWith (pkgs // packages);
      packages = {
        gitlint = callPackage ./lint { };
        gitreview = callPackage ./review { };
      };
    in
      {
        devShell = callPackage ./shell.nix { };
      }));
}
