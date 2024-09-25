module github.com/peterldowns/pgtestdb/migrators/goosemigrator

go 1.18

replace github.com/peterldowns/pgtestdb => ../../

require (
	github.com/jackc/pgx/v5 v5.5.5
	github.com/peterldowns/pgtestdb v0.0.15
	github.com/peterldowns/testy v0.0.1
	github.com/pressly/goose/v3 v3.11.2
)

require (
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20231201235250-de7065d80cb9 // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	golang.org/x/crypto v0.23.0 // indirect
	golang.org/x/exp v0.0.0-20230626212559-97b1e661b5df // indirect
	golang.org/x/sync v0.2.0 // indirect
	golang.org/x/text v0.15.0 // indirect
	golang.org/x/tools v0.9.1 // indirect
)
