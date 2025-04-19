{buildGoModule}:
buildGoModule {
  name = "git-review";
  pname = "git-review";
  src = ./.;

  vendorHash = "sha256-TeT0+wqKMdoHdGOBu+8Q/fGjm7AXxn3xAXUsvTffmmU=";

  postInstall = ''
    mv $out/bin/review $out/bin/git-review
  '';

  meta.mainProgram = "git-review";
}
