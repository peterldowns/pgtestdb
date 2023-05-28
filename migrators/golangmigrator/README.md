# golangmigrator

golangmigrator provides a migrator that can be used out of the box with projects
that use [golang-migrate/migrate](https://github.com/golang-migrate/migrate) for
migrations. Because we must be able to hash the migrations to create a unique
template database, this implementation does not support other migration
sources..

You can configure the migrations directory, and the filesystem being used.
Here's an example:

```go
//go:embed migrations/*.sql
var exampleFS embed.FS

func TestMigrateFromEmbeddedFS(t *testing.T) {
	t.Parallel()
	gm := golangmigrator.New(
		"migrations",
		golangmigrator.WithFS(exampleFS),
	)

	db := testdb.New(t, testdb.Config{
		Host:     "localhost",
		User:     "postgres",
		Password: "password",
		Port:     "5433",
		Options:  "sslmode=disable",
	}, gm)
	assert.NotEqual(t, nil, db)
}

func TestMigrateFromDisk(t *testing.T) {
	t.Parallel()
	gm := golangmigrator.New("migrations")
	db := testdb.New(t, testdb.Config{
		Host:     "localhost",
		User:     "postgres",
		Password: "password",
		Port:     "5433",
		Options:  "sslmode=disable",
	}, gm)
	assert.NotEqual(t, nil, db)
}
```