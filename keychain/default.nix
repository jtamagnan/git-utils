{buildGoModule}:
buildGoModule {
  name = "git-keychain";
  pname = "git-keychain";
  src = ./.;

  vendorHash = "sha256-LI/OTwtHOh7HQ7z/xUCBzkA1j0odgb1PWUfJEd0gWA0=";

  postInstall = ''
    mv $out/bin/keychain $out/bin/git-keychain
  '';

  meta.mainProgram = "git-keychain";
}