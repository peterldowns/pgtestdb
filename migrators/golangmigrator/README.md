# golangmigrator

```
go get github.com/peterldowns/pgtestdb/migrators/golangmigrator@latest
```

golangmigrator provides a migrator that can be used out of the box with projects
that use [golang-migrate/migrate](https://github.com/golang-migrate/migrate) for
migrations.

Because `Hash()` requires calculating a unique hash based on the contents of
the migrations, this implementation only supports reading migration
files from disk or an embedded filesystem.

You can configure the migrations directory and the filesystem being used.
Here's an example:

```go
//go:embed migrations/*.sql
var exampleFS embed.FS

func TestMigrateFromEmbeddedFS(t *testing.T) {
  gm := golangmigrator.New(
    "migrations",
    golangmigrator.WithFS(exampleFS),
  )

  db := pgtestdb.New(t, pgtestdb.Config{
    Host:     "localhost",
    User:     "postgres",
    Password: "password",
    Port:     "5433",
    Options:  "sslmode=disable",
  }, gm)
  assert.NotEqual(t, nil, db)
}

func TestMigrateFromDisk(t *testing.T) {
  gm := golangmigrator.New("migrations")
  db := pgtestdb.New(t, pgtestdb.Config{
    Host:     "localhost",
    User:     "postgres",
    Password: "password",
    Port:     "5433",
    Options:  "sslmode=disable",
  }, gm)
  assert.NotEqual(t, nil, db)
}
```
