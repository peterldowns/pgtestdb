{
  description = "pgtestdb makes it cheap and easy to create ephemeral Postgres databases for your golang tests.";
  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs";

    flake-utils.url = "github:numtide/flake-utils";

    flake-compat.url = "github:edolstra/flake-compat";
    flake-compat.flake = false;
  };

  outputs = { ... }@inputs:
    inputs.flake-utils.lib.eachDefaultSystem
      (system:
        let
          overlays = [ ];
          pkgs = import inputs.nixpkgs {
            inherit system overlays;
          };
          lib = pkgs.lib;
          version = (builtins.readFile ./VERSION);
        in
        rec {
          packages = rec { };
          apps = rec { };
          devShells = rec {
            default = pkgs.mkShell {
              buildInputs = [ ];
              packages = with pkgs; [
                # Go
                delve
                go-outline
                go
                golangci-lint
                gopkgs
                gopls
                gotools
                # Python
                python311Full
                black
                ruff
                # Nix
                rnix-lsp
                nixpkgs-fmt
                # Other
                just
                postgresql
                docker
              ];

              shellHook = ''
                # The path to this repository
                shell_nix="''${IN_LORRI_SHELL:-$(pwd)/shell.nix}"
                workspace_root=$(dirname "$shell_nix")
                export WORKSPACE_ROOT="$workspace_root"

                # Puts the $GOPATH/$GOCACHE/$GOENV in $TOOLCHAIN_ROOT,
                # and ensures that the GOPATH's bin dir is on the PATH so tools
                # can be installed with `go install`.
                #
                # Any tools installed explicitly with `go install` will take precedence
                # over versions installed by Nix due to the ordering here.
                #
                # Puts the toolchain folder adjacent to the repo so that tools
                # running inside the repo don't ever scan its contents.
                export TOOLCHAIN_NAME=".toolchain-$(basename $WORKSPACE_ROOT)"
                export TOOLCHAIN_ROOT="$(dirname $WORKSPACE_ROOT)/$TOOLCHAIN_NAME"
                export GOROOT=
                export GOCACHE="$TOOLCHAIN_ROOT/go/cache"
                export GOENV="$TOOLCHAIN_ROOT/go/env"
                export GOPATH="$TOOLCHAIN_ROOT/go/path"
                export GOMODCACHE="$GOPATH/pkg/mod"
                export PATH=$(go env GOPATH)/bin:$PATH
                export CGO_ENABLED=1

                # Make it easy to test while developing; add the golang and nix
                # build outputs to the path.
                export PATH="$workspace_root/bin:$workspace_root/result/bin:$PATH"
              '';

              # Need to disable fortify hardening because GCC is not built with -oO,
              # which means that if CGO_ENABLED=1 (which it is by default) then the golang
              # debugger fails.
              # see https://github.com/NixOS/nixpkgs/pull/12895/files
              hardeningDisable = [ "fortify" ];
            };
          };
        }
      );
}
