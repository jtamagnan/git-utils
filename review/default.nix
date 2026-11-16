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

  vendorHash = "sha256-oyHjqrCI6mnuVgPhd9Xgmgl7igw+9Y3Zl9hKNUtElxc=";

  # Tests require external dependencies.
  doCheck = false;

  # buildGoModule names the binary after the module (`review`); the
  # tool is expected to be `git-review` on PATH.
  postInstall = ''
    mv $out/bin/review $out/bin/git-review
  '';

  meta.mainProgram = "git-review";
}
