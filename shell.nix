{
  mkShell,
  go,
  gopls,
  golangci-lint,
  pre-commit,
  prek,
  # gitlint,
}:
mkShell {
  packages = [
    go
    gopls
    golangci-lint
    pre-commit
    prek
    # gitlint
  ];
}
