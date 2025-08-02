{stdenv, go}:
stdenv.mkDerivation {
  name = "git-review";
  pname = "git-review";
  src = ../.;  # Use parent directory to include all modules

  nativeBuildInputs = [ go ];

  buildPhase = ''
    runHook preBuild
    cd review
    export HOME=$(mktemp -d)
    go build -o git-review
    runHook postBuild
  '';

  installPhase = ''
    runHook preInstall
    mkdir -p $out/bin
    cp git-review $out/bin/
    runHook postInstall
  '';

  # Disable tests since they require external dependencies
  doCheck = false;

  meta.mainProgram = "git-review";
}
