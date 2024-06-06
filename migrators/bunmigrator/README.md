# bunmigrator

```
go get github.com/peterldowns/pgtestdb/migrators/bunmigrator@latest
```

bunmigrator provides a migrator that can be used with projects that make use of [uptrace/bun](https://github.com/uptrace/bun) for migrations.

Because `Hash()` requires calculating a unique hash based on the contents of migrations, this implementation only supports reading migration files from disk or an embedded filesystem.

You can configure the migrations directory and the filesystem being used.
Here's an example:

```go
//go:embed migrations/*.sql
var exampleFS embed.FS

func TestMigrateFromEmbeddedFS(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	bm := bunmigrator.New("migrations", bunmigrator.WithFS(exampleFS))
	db := pgtestdb.New(t, pgtestdb.Config{
		DriverName: "pg",
		Host:       "localhost",
		User:       "postgres",
		Password:   "password",
		Port:       "5433",
		Options:    "sslmode=disable",
	}, bm)

	assert.NotEqual(t, nil, db)
}

func TestMigrateFromDisk(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	bm := bunmigrator.New("migrations")
	db := pgtestdb.New(t, pgtestdb.Config{
		DriverName: "pg",
		Host:       "localhost",
		User:       "postgres",
		Password:   "password",
		Port:       "5433",
		Options:    "sslmode=disable",
	}, bm)

	assert.NotEqual(t, nil, db)
}
```