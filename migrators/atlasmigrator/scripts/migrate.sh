#!/usr/bin/env sh
# Should be run as ./scripts/migrate.sh
source ./scripts/env.sh
atlas schema apply --url "$ATLAS_URL" --to 'file://schema.hcl'
