{ buildGoModule }:
buildGoModule {
  pname = "git-review";
  version = "0.1.0";

  src = ../.;           # include all modules (review replaces sibling modules)
  modRoot = "review";   # build the review module

  # Only build the main command, not the library subpackages.
  subPackages = [ "." ];

  # Use module mode, not go.work workspace mode, so the local
  # `replace` directives in review/go.mod are honoured.
  env.GOWORK = "off";

  # proxyVendor: only third-party modules go in the fixed-output
  # derivation. Local `replace => ../x` modules are read from src, so
  # edits to sibling modules are picked up without bumping vendorHash.
  proxyVendor = true;
  vendorHash = "sha256-ym7FJAlR1NumwDDoXyCSyM68d/OJJnDVrCAES06hnoE=";

  # Tests require external dependencies.
  doCheck = false;

  # buildGoModule names the binary after the module (`review`); the
  # tool is expected to be `git-review` on PATH.
  postInstall = ''
    mv $out/bin/review $out/bin/git-review
  '';

  meta.mainProgram = "git-review";
}
