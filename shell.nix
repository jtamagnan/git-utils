{
  mkShell,
  go,
  gopls,
  golangci-lint,
  pre-commit,
  # gitlint,
}:
mkShell {
  packages = [
    go
    gopls
    golangci-lint
    pre-commit
    # gitlint
  ];
}
