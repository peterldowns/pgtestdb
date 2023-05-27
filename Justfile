# This Justfile contains rules/targets/scripts/commands that are used when
# developing. Unlike a Makefile, running `just <cmd>` will always invoke
# that command. For more information, see https://github.com/casey/just
#
#
# this setting will allow passing arguments through to tasks, see the docs here
# https://just.systems/man/en/chapter_24.html#positional-arguments
set positional-arguments

# print all available commands by default
default:
  just --list

# run the test suite
test *args='./...':
  go test "$@"

# run the tests for all subpackages
test-all:
  #!/usr/bin/env bash
  go test ./...
  for migrator in ./migrators/*; do
    go test $migrator/...
  done

# lint the entire codebase
lint *args:
  golangci-lint run --fix --config .golangci.yaml "$@"
  find . -name '*.nix' | xargs nixpkgs-fmt
