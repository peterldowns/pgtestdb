#!/usr/bin/env sh
# Should be run as ./scripts/createdb.sh
source ./scripts/env.sh
psql "$PG_URL" -c "CREATE ROLE atlas CREATEDB PASSWORD 'password' LOGIN;"
psql "$PG_URL" -c "CREATE DATABASE atlas WITH OWNER atlas;"
psql "$ATLAS_URL" -c "select current_time;"
