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

# test pgtestdb
test *args='./...':
  go test -race "$@"

# test pgtestdb + migrators
test-all:
  #!/usr/bin/env bash
  go test -race ./... ./migrators/*/

# lint pgtestdb
lint *args:
  golangci-lint run --fix --config .golangci.yaml "$@"

# lint pgtestdb + migrators
lint-all:
  golangci-lint run --fix --config .golangci.yaml ./migrators/*/

# lint nix files
lint-nix:
  find . -name '*.nix' | xargs nixpkgs-fmt

tidy:
  #!/usr/bin/env bash
  go mod tidy
  for subdir in ./migrators/*/; do
    pushd $subdir
    go mod tidy
    popd
  done
  go mod tidy
  rm -f go.work.sum
  go work sync
  go mod tidy

  

# tag testdb
tag:
  #!/usr/bin/env bash
  set -e
  raw="v$(cat VERSION)"
  git tag "$raw"
  # commit="${raw}+commit.$(git rev-parse --short HEAD)"
  # git tag "$commit"

# tag migrators
tag-migrators:
  #!/usr/bin/env bash
  set -e
  raw="v$(cat VERSION)"
  git tag "migrators/atlasmigrator/$raw"
  git tag "migrators/dbmigrator/$raw"
  git tag "migrators/golangmigrator/$raw"
  git tag "migrators/goosemigrator/$raw"
  git tag "migrators/sqlmigrator/$raw"
  # commit="${raw}+commit.$(git rev-parse --short HEAD)"
  # git tag "migrators/atlasmigrator/$commit"
  # git tag "migrators/dbmigrator/$commit"
  # git tag "migrators/golangmigrator/$commit"
  # git tag "migrators/goosemigrator/$commit"
  # git tag "migrators/sqlmigrator/$commit"
