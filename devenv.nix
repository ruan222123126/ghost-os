{ pkgs, ... }:

{
  packages = [
    pkgs.cargo-udeps
    pkgs.gcc
    pkgs.golangci-lint
    pkgs.go
    pkgs.nodejs_22
    pkgs.openssl
    pkgs.pkg-config
    pkgs.pnpm
    pkgs.rustup
  ];

  enterShell = ''
    echo "Ghost-OS dev checks:"
    echo "  python task.py scavenge"
    echo "  python task.py go-lint"
    echo "  python task.py rust-clippy"
    echo "  python task.py web-knip"
  '';
}
