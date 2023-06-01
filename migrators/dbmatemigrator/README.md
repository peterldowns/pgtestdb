# dbmatemigrator

```shell
go get github.com/peterldowns/pgtestdb/migrators/dbmatemigrator@latest
```

dbmatemigrator provides a migrator that can be used out of the box with projects
that use [amacneil/dbmate](https://github.com/amacneil/dbmate) for migrations.

You can configure the migrations directory, the table name, and the filesystem
being used. Here's an example:

```go
//go:embed migrations/*.sql more/*.sql
var migrationsFS embed.FS

func TestDbmateMigratorWithFSAndOptions(t *testing.T) {
  m := dbmatemigrator.New(
    // Use the embedded filesystem
    dbmatemigrator.WithFS(migrationsFS),
    // Use migrations in the "migrations" and "more" directories
    dbmatemigrator.WithDir("migrations", "more"),
    // Use "dbmate_migrations_example" as the name of the table in which to
    // store records about which migrations are applied.
    dbmatemigrator.WithTableName("dbmate_migrations_example"),
  )
  db := testdb.New(t, testdb.Config{
    DriverName: "pgx",
    Host:       "localhost",
    User:       "postgres",
    Password:   "password",
    Port:       "5433",
    Options:    "sslmode=disable",
  }, m)
  assert.NotEqual(t, nil, db)
}

func TestDbmateMigratorWithDefaults(t *testing.T) {
  // If you're using the default settings, you don't need to pass any options.
  // This will read migrations from disk, from the folder "./db/migrations",
  // and store the results in the "schema_migrations" table.
  m := dbmatemigrator.New()
  db := testdb.New(t, testdb.Config{
    DriverName: "pgx",
    Host:       "localhost",
    User:       "postgres",
    Password:   "password",
    Port:       "5433",
    Options:    "sslmode=disable",
  }, m)
  assert.NotEqual(t, nil, db)
}
```