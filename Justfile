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
  go test -race "$@"

# run the tests for all subpackages
test-all:
  #!/usr/bin/env bash
  go test -race ./... ./migrators/**

# lint the entire codebase
lint *args:
  golangci-lint run --fix --config .golangci.yaml "$@"
  find . -name '*.nix' | xargs nixpkgs-fmt

tag:
  #!/usr/bin/env bash
  set -e
  raw="v$(cat VERSION)"
  commit="${raw}+commit.$(git rev-parse --short HEAD)"
  git tag "$raw"
  git tag "$commit"
  git tag "migrators/atlasmigrator/$raw"
  git tag "migrators/atlasmigrator/$commit"
  git tag "migrators/dbmigrator/$raw"
  git tag "migrators/dbmigrator/$commit"
  git tag "migrators/golangmigrator/$raw"
  git tag "migrators/golangmigrator/$commit"
  git tag "migrators/goosemigrator/$raw"
  git tag "migrators/goosemigrator/$commit"
  git tag "migrators/sqlmigrator/$raw"
  git tag "migrators/sqlmigrator/$commit"
