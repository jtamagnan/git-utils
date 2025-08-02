{stdenv, go}:
stdenv.mkDerivation {
  name = "git-lint";
  src = ../.;  # Use parent directory to include all modules

  nativeBuildInputs = [ go ];

  buildPhase = ''
    runHook preBuild
    cd lint
    export HOME=$(mktemp -d)
    go build -o git-lint
    runHook postBuild
  '';

  installPhase = ''
    runHook preInstall
    mkdir -p $out/bin
    cp git-lint $out/bin/
    runHook postInstall
  '';

  # Disable tests since they require git to be available
  doCheck = false;

  meta.mainProgram = "git-lint";
}
