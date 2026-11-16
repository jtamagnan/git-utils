{ buildGoModule }:
buildGoModule {
  pname = "git-lint";
  version = "0.1.0";

  src = ../.;           # include all modules (lint replaces ../git)
  modRoot = "lint";     # build the lint module

  # Only build the main command, not the local ./lib sub-module.
  subPackages = [ "." ];

  # Use module mode, not go.work workspace mode, so the
  # `replace => ../git` directive in lint/go.mod is honoured.
  env.GOWORK = "off";

  # proxyVendor: only third-party modules go in the fixed-output
  # derivation. Local `replace => ../x` modules are read from src, so
  # edits to sibling modules are picked up without bumping vendorHash.
  proxyVendor = true;
  vendorHash = "sha256-lPnSTqtfwKpRg4XBgUIj0niPzF1PEQDdcZCLOppuMCA=";

  # Tests require a git repo to be available.
  doCheck = false;

  # buildGoModule names the binary after the module (`lint`); the
  # tool is expected to be `git-lint` on PATH.
  postInstall = ''
    mv $out/bin/lint $out/bin/git-lint
  '';

  meta.mainProgram = "git-lint";
}
