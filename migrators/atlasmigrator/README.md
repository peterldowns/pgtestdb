# atlasmigrator

atlasmigrator provides migrators that can be used out of the box with projects that use [ariga/atlas](https://github.com/atlas) for migrations.

As [recommended by the Atlas maintainers](https://github.com/ariga/atlas/issues/1527#issuecomment-1465123713), these migrators expect the `atlas` CLI program to exist on your path at the time that the tests are run, and shells out to that program to run migrations. These migrators do *not* call the Atlas golang code directly.

## DirMigrator for Versioned Workflows

The `DirMigrator` runs migrations by calling `atlas migrate apply`:

```shell
atlas migrate apply \
    --url "$DB" \
    --dir "file://$migrationsDirPath"
```

where `migrationsDirPath` is the path to a folder full
of migration files as described [in the Atlas documentation for "Versioned Workflows"](https://atlasgo.io/versioned/apply).

You can use it like this:

```go
func TestWithchemaMigrator(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	m := atlasmigrator.NewSchemaMigrator("schema.hcl")
	db := testdb.New(t, testdb.Config{
		Host:     "localhost",
		User:     "postgres",
		Password: "password",
		Port:     "5433",
		Options:  "sslmode=disable",
	}, m)
	assert.NotEqual(t, nil, db)
}
```

## SchemaMigrator for Declarative Workflows

The `SchemaMigrator` runs migrations by calling `atlas schema apply`:

```shell
atlas schema apply \
    --auto-approve
    --url "$DB" \
    --to "file://$schemaFilePath"
```

where `schemaFilePath` is the path to a `.hcl` schema file as described [in the Atlas documented for "Declarative Workflows"](https://atlasgo.io/declarative/apply).

You can use it like this:
```go
func TestWithchemaMigrator(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	m := atlasmigrator.NewSchemaMigrator("schema.hcl")
	db := testdb.New(t, testdb.Config{
		Host:     "localhost",
		User:     "postgres",
		Password: "password",
		Port:     "5433",
		Options:  "sslmode=disable",
	}, m)
	assert.NotEqual(t, nil, db)
}
```