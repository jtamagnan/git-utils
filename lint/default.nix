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

  vendorHash = "sha256-Ay7a8XJs4ldZGdILLAQt7VsZHhyX7ZDDzs72JXA9l0s=";

  # Tests require a git repo to be available.
  doCheck = false;

  # buildGoModule names the binary after the module (`lint`); the
  # tool is expected to be `git-lint` on PATH.
  postInstall = ''
    mv $out/bin/lint $out/bin/git-lint
  '';

  meta.mainProgram = "git-lint";
}
