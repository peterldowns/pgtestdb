package sqlmigrator_test

import (
	"context"
	"embed"
	"testing"

	migrate "github.com/rubenv/sql-migrate"

	"github.com/peterldowns/testy/assert"
	"github.com/peterldowns/testy/check"

	"github.com/peterldowns/testdb"
	"github.com/peterldowns/testdb/migrators/sqlmigrator"
)

func TestSQLMigratorFromFolder(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	migrations := &migrate.FileMigrationSource{
		Dir: "migrations",
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

	assert.NoFailures(t, func() {
		var lastAppliedMigration string
		err := db.QueryRowContext(ctx, "select id from example_schema.example_sql_migrations order by applied_at desc limit 1").Scan(&lastAppliedMigration)
		assert.Nil(t, err)
		check.Equal(t, "0002_cats.sql", lastAppliedMigration)
	})

	var numUsers int
	err := db.QueryRowContext(ctx, "select count(*) from users;").Scan(&numUsers)
	assert.Nil(t, err)
	check.Equal(t, 0, numUsers)

	var numCats int
	err = db.QueryRowContext(ctx, "select count(*) from cats;").Scan(&numCats)
	assert.Nil(t, err)
	check.Equal(t, 0, numCats)

	var numBlogPosts int
	err = db.QueryRowContext(ctx, "select count(*) from blog_posts;").Scan(&numBlogPosts)
	assert.Nil(t, err)
	check.Equal(t, 0, numBlogPosts)
}

//go:embed migrations/*.sql
var exampleFS embed.FS

func TestSQLMigratorFromFS(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

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

	assert.NoFailures(t, func() {
		var lastAppliedMigration string
		err := db.QueryRowContext(ctx, "select id from example_schema.example_sql_migrations order by applied_at desc limit 1").Scan(&lastAppliedMigration)
		assert.Nil(t, err)
		check.Equal(t, "0002_cats.sql", lastAppliedMigration)
	})

	var numUsers int
	err := db.QueryRowContext(ctx, "select count(*) from users;").Scan(&numUsers)
	assert.Nil(t, err)
	check.Equal(t, 0, numUsers)

	var numCats int
	err = db.QueryRowContext(ctx, "select count(*) from cats;").Scan(&numCats)
	assert.Nil(t, err)
	check.Equal(t, 0, numCats)

	var numBlogPosts int
	err = db.QueryRowContext(ctx, "select count(*) from blog_posts;").Scan(&numBlogPosts)
	assert.Nil(t, err)
	check.Equal(t, 0, numBlogPosts)
}
