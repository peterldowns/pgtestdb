#!/usr/bin/env sh
# Should be run as ./scripts/diff.sh
source ./scripts/env.sh
atlas schema diff --to "$ATLAS_URL" --from 'file://schema.hcl' --dev-url 'docker://postgres/15' --format '{{ hcl . " " }}'
