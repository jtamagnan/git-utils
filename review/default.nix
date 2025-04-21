{buildGoModule}:
buildGoModule {
  name = "git-review";
  pname = "git-review";
  src = ./.;

  vendorHash = "sha256-17WFu9GA7yh5Fzws4U7xqoC0t2Ox94WrtbDzmialtls=";

  postInstall = ''
    mv $out/bin/review $out/bin/git-review
  '';

  meta.mainProgram = "git-review";
}
