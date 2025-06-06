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
test-all *args='':
  #!/usr/bin/env bash
  go test -race github.com/peterldowns/pgtestdb/... "$@"

# lint pgtestdb
lint *args:
  golangci-lint run --fix --config .golangci.yaml "$@"

# lint pgtestdb + migrators
lint-all:
  golangci-lint run --fix --config .golangci.yaml ./ ./migrators/*/

# lint nix files
lint-nix:
  find . -name '*.nix' | xargs nixpkgs-fmt

# (attempt) to tidy all go.mod files
tidy:
  #!/usr/bin/env bash
  go mod tidy -go=1.21.0 -compat=1.21.0
  for subdir in ./migrators/*/; do
    pushd $subdir
    go mod tidy -go=1.21.0 -compat=1.21.0
    popd
  done
  rm -f go.work.sum
  go mod tidy -go=1.21.0 -compat=1.21.0
  go work sync
  go mod tidy -go=1.21.0 -compat=1.21.0

# tag pgtestdb with current version
tag:
  #!/usr/bin/env bash
  set -e
  raw="$(cat VERSION)"
  git tag "$raw"
  # commit="${raw}+commit.$(git rev-parse --short HEAD)"
  # git tag "$commit"

# tag migrators with current version.
tag-migrators:
  #!/usr/bin/env bash
  set -e
  raw="$(cat VERSION)"
  git tag "migrators/pgmigrator/$raw"
  git tag "migrators/golangmigrator/$raw"
  git tag "migrators/goosemigrator/$raw"
  git tag "migrators/dbmatemigrator/$raw"
  git tag "migrators/atlasmigrator/$raw"
  git tag "migrators/sqlmigrator/$raw"
  git tag "migrators/bunmigrator/$raw"
  git tag "migrators/ternmigrator/$raw"

goproxy-release:
  #!/usr/bin/env bash
  set -e
  export GOPROXY=proxy.golang.org
  version="$(cat VERSION)"
  go list -m github.com/peterldowns/pgtestdb@${version}
  go list -m github.com/peterldowns/pgtestdb/migrators/pgmigrator@${version}
  go list -m github.com/peterldowns/pgtestdb/migrators/golangmigrator@${version}
  go list -m github.com/peterldowns/pgtestdb/migrators/goosemigrator@${version}
  go list -m github.com/peterldowns/pgtestdb/migrators/dbmatemigrator@${version}
  go list -m github.com/peterldowns/pgtestdb/migrators/atlasmigrator@${version}
  go list -m github.com/peterldowns/pgtestdb/migrators/sqlmigrator@${version}
  go list -m github.com/peterldowns/pgtestdb/migrators/bunmigrator@${version}
  go list -m github.com/peterldowns/pgtestdb/migrators/ternmigrator@${version}

# set the VERSION and go.mod versions.
bump-version version:
  #!/usr/bin/env bash
  OLD_VERSION=$(cat VERSION)
  NEW_VERSION=$1
  echo "bumping $OLD_VERSION -> $NEW_VERSION"
  echo $NEW_VERSION > VERSION
  sed -i -e "s/$OLD_VERSION/$NEW_VERSION/g" README.md
  sed -i -e "s,github.com/peterldowns/pgtestdb $OLD_VERSION,github.com/peterldowns/pgtestdb $NEW_VERSION,g" migrators/*/go.mod
