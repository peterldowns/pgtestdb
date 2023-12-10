module github.com/peterldowns/pgtestdb/migrators/sqlmigrator

go 1.18

replace github.com/peterldowns/pgtestdb => ../../

require (
	github.com/jackc/pgx/v5 v5.5.1
	github.com/peterldowns/pgtestdb v0.0.12
	github.com/peterldowns/testy v0.0.1
	github.com/rubenv/sql-migrate v1.4.0
)

require (
	github.com/go-gorp/gorp/v3 v3.1.0 // indirect
	github.com/go-sql-driver/mysql v1.7.1 // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/mattn/go-sqlite3 v1.14.16 // indirect
	github.com/sirupsen/logrus v1.9.2 // indirect
	golang.org/x/crypto v0.11.0 // indirect
	golang.org/x/exp v0.0.0-20230626212559-97b1e661b5df // indirect
	golang.org/x/sync v0.2.0 // indirect
	golang.org/x/text v0.11.0 // indirect
)
