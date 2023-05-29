# sqlmigrator

```shell
go get github.com/peterldowns/testdb/migrators/sqlmigrator@latest
```

sqlmigrator provides migrators that can be used out of the box with projects that use [rubenv/sql-migrate](https://github.com/rubenv/sql-migrate) for migrations.

sqlmigrator supports any migration source and any configuration settings allowed with sql-migrate. Instead of using the global migration instance,
you [pass in a `migrate.MigrationSet`, which means that this is parallel/concurrency safe](https://github.com/rubenv/sql-migrate/issues/226#issuecomment-1268127309).

You can configure the migrations directory, the table name, and the filesystem
being used. Here's an example:

```go
func TestSQLMigratorFromDisk(t *testing.T) {
	migrations := &migrate.FileMigrationSource{Dir: "migrations"}
	sm := sqlmigrator.New(migrations, &migrate.MigrationSet{})
	db := testdb.New(t, testdb.Config{
		Host:     "localhost",
		User:     "postgres",
		Password: "password",
		Port:     "5433",
		Options:  "sslmode=disable",
	}, sm)
	assert.NotEqual(t, nil, db)
}

//go:embed migrations/*.sql
var exampleFS embed.FS

func TestSQLMigratorFromFSWithSomeConfiguration(t *testing.T) {
	migrations := &migrate.EmbedFileSystemMigrationSource{
		FileSystem: exampleFS,
		Root:       "migrations",
	}
	sm := sqlmigrator.New(migrations, &migrate.MigrationSet{
		TableName:  "example_sql_migrations",
		SchemaName: "example_schema",
	})
	db := testdb.New(t, testdb.Config{
		Host:     "localhost",
		User:     "postgres",
		Password: "password",
		Port:     "5433",
		Options:  "sslmode=disable",
	}, sm)
	assert.NotEqual(t, nil, db)
}
```