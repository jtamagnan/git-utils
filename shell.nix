{
  mkShell,
  go,
  gopls,
  gitlint,
}:
mkShell {
  packages = [
    go
    gopls
    gitlint
  ];
}
