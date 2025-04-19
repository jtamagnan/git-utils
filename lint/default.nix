{buildGoModule}:
buildGoModule {
  name = "git-lint";
  src = ./.;

  vendorHash = "sha256-TeT0+wqKMdoHdGOBu+8Q/fGjm7AXxn3xAXUsvTffmmU=";

  postInstall = ''
    mv $out/bin/lint $out/bin/git-lint
  '';

  meta.mainProgram = "git-lint";
}
