module github.com/peterldowns/pgtestdb/migrators/atlasmigrator

go 1.21.0

toolchain go1.22.1

replace github.com/peterldowns/pgtestdb => ../../

require (
	github.com/jackc/pgx/v5 v5.7.1
	github.com/peterldowns/pgtestdb v0.1.1
	github.com/peterldowns/testy v0.0.1
)

require (
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	golang.org/x/crypto v0.27.0 // indirect
	golang.org/x/exp v0.0.0-20240325151524-a685a6edb6d8 // indirect
	golang.org/x/sync v0.8.0 // indirect
	golang.org/x/text v0.18.0 // indirect
)
