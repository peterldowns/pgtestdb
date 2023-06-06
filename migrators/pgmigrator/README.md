# pgmigrator

```shell
go get github.com/peterldowns/pgtestdb/migrators/pgmigrator@latest
```

pgmigrator provides a migrator that can be used out of the box with projects that use [peterldowns/pgmigrate](https://github.com/peterldowns/pgmigrate) for migrations.

You can use migrations from disk or from an embedded FS, and you can set the table name that the migration records are stored in. Here's an example:

```go
func TestPGMigratorFromDisk(t *testing.T) {
	dir := os.DirFS("migrations")
	pgm, err := pgmigrator.New(
    dir,
    pgmigrator.WithTableName("example_table_name")),
  )
	assert.Nil(t, err)
	db := pgtestdb.New(t, pgtestdb.Config{
		DriverName: "pgx",
		Host:       "localhost",
		User:       "postgres",
		Password:   "password",
		Port:       "5433",
		Options:    "sslmode=disable",
	}, pgm)
	assert.NotEqual(t, nil, db)
}

//go:embed *.sql
var exampleFS embed.FS

func TestPGMigratorFromFS(t *testing.T) {
	pgm, err := pgmigrator.New(exampleFS)
	assert.Nil(t, err)
	db := pgtestdb.New(t, pgtestdb.Config{
		DriverName: "pgx",
		Host:       "localhost",
		User:       "postgres",
		Password:   "password",
		Port:       "5433",
		Options:    "sslmode=disable",
	}, pgm)
	assert.NotEqual(t, nil, db)
}
```
