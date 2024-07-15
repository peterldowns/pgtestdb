# goosemigrator

```shell
go get github.com/peterldowns/pgtestdb/migrators/goosemigrator@latest
```

goosemigrator provides migrators that can be used out of the box with projects
that use [pressly/goose](https://github.com/pressly/goose) for migrations.

Currently only [SQL migrations](https://github.com/pressly/goose#sql-migrations)
are supported, and can be read from a directory on disk or from an embedded
filesystem.
[Golang-defined migrations](https://github.com/pressly/goose#go-migrations) are
not supported by default.

[You can configure the migrations directory, the table name, and the filesystem
being used. Here's an example:]()

```go
func TestGooseMigratorFromDisk(t *testing.T) {
  m := goosemigrator.New("migrations")
  db := pgtestdb.New(t, pgtestdb.Config{
    DriverName: "pgx",
    Host:       "localhost",
    User:       "postgres",
    Password:   "password",
    Port:       "5433",
    Options:    "sslmode=disable",
  }, m)
  assert.NotEqual(t, nil, db)
}

//go:embed migrations/*.sql
var exampleFS embed.FS

func TestGooseMigratorFromFS(t *testing.T) {
  gm := goosemigrator.New(
    "migrations",
    goosemigrator.WithFS(exampleFS),
    goosemigrator.WithTableName("goose_example_migrations"),
  )
  db := pgtestdb.New(t, pgtestdb.Config{
    DriverName: "pgx",
    Host:       "localhost",
    User:       "postgres",
    Password:   "password",
    Port:       "5433",
    Options:    "sslmode=disable",
  }, gm)
  assert.NotEqual(t, nil, db)
}
```
