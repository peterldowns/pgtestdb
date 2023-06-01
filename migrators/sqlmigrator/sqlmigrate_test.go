package sqlmigrator_test

import (
	"context"
	"embed"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib" // "pgx" driver
	"github.com/peterldowns/testy/assert"
	"github.com/peterldowns/testy/check"

	migrate "github.com/rubenv/sql-migrate"

	"github.com/peterldowns/pgtestdb"
	"github.com/peterldowns/pgtestdb/migrators/sqlmigrator"
)

func TestSQLMigratorFromDisk(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	sm := sqlmigrator.New(&migrate.FileMigrationSource{
		Dir: "migrations",
	}, nil)
	db := testdb.New(t, testdb.Config{
		DriverName: "pgx",
		Host:       "localhost",
		User:       "postgres",
		Password:   "password",
		Port:       "5433",
		Options:    "sslmode=disable",
	}, sm)
	assert.NotEqual(t, nil, db)

	assert.NoFailures(t, func() {
		var lastAppliedMigration string
		err := db.QueryRowContext(ctx, "select id from public.gorp_migrations order by applied_at desc limit 1").Scan(&lastAppliedMigration)
		assert.Nil(t, err)
		check.Equal(t, "0002_cats.sql", lastAppliedMigration)
	})

	var numUsers int
	err := db.QueryRowContext(ctx, "select count(*) from users").Scan(&numUsers)
	assert.Nil(t, err)
	check.Equal(t, 0, numUsers)

	var numCats int
	err = db.QueryRowContext(ctx, "select count(*) from cats").Scan(&numCats)
	assert.Nil(t, err)
	check.Equal(t, 0, numCats)

	var numBlogPosts int
	err = db.QueryRowContext(ctx, "select count(*) from blog_posts").Scan(&numBlogPosts)
	assert.Nil(t, err)
	check.Equal(t, 0, numBlogPosts)
}

//go:embed migrations/*.sql
var exampleFS embed.FS

func TestSQLMigratorFromFS(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	sm := sqlmigrator.New(
		&migrate.EmbedFileSystemMigrationSource{
			FileSystem: exampleFS,
			Root:       "migrations",
		},
		&migrate.MigrationSet{
			SchemaName: "altschema",
			TableName:  "alt_migrations_table_name",
		},
	)
	db := testdb.New(t, testdb.Config{
		DriverName: "pgx",
		Host:       "localhost",
		User:       "postgres",
		Password:   "password",
		Port:       "5433",
		Options:    "sslmode=disable",
	}, sm)
	assert.NotEqual(t, nil, db)

	assert.NoFailures(t, func() {
		var lastAppliedMigration string
		err := db.QueryRowContext(ctx, "select id from altschema.alt_migrations_table_name order by applied_at desc limit 1").Scan(&lastAppliedMigration)
		assert.Nil(t, err)
		check.Equal(t, "0002_cats.sql", lastAppliedMigration)
	})

	var numUsers int
	err := db.QueryRowContext(ctx, "select count(*) from users").Scan(&numUsers)
	assert.Nil(t, err)
	check.Equal(t, 0, numUsers)

	var numCats int
	err = db.QueryRowContext(ctx, "select count(*) from cats").Scan(&numCats)
	assert.Nil(t, err)
	check.Equal(t, 0, numCats)

	var numBlogPosts int
	err = db.QueryRowContext(ctx, "select count(*) from blog_posts").Scan(&numBlogPosts)
	assert.Nil(t, err)
	check.Equal(t, 0, numBlogPosts)
}

// DisableCreateTable is broken for MigrationSet instances, see
// https://github.com/rubenv/sql-migrate/pull/242 Once that PR is merged
// and a new release is created, I can bump this repository's
// dependencies and uncomment this test.
/*
func TestSQLMigratorWithTableDisabled(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	sm := sqlmigrator.New(
		&migrate.EmbedFileSystemMigrationSource{
			FileSystem: exampleFS,
			Root:       "migrations",
		}, &migrate.MigrationSet{
			DisableCreateTable: true,
			SchemaName:         "public",
			TableName:          "broken",
		},
	)
	db := testdb.New(t, testdb.Config{
		DriverName: "pgx",
		Host:     "localhost",
		User:     "postgres",
		Password: "password",
		Port:     "5433",
		Options:  "sslmode=disable",
	}, sm)
	assert.NotEqual(t, nil, db)

	// This table should not exist, so the query should fail.
	var lastAppliedMigration string
	err := db.QueryRowContext(ctx, "select id from public.broken order by applied_at desc limit 1").Scan(&lastAppliedMigration)
	assert.Error(t, err)

	var numUsers int
	err := db.QueryRowContext(ctx, "select count(*) from users").Scan(&numUsers)
	assert.Nil(t, err)
	check.Equal(t, 0, numUsers)

	var numCats int
	err = db.QueryRowContext(ctx, "select count(*) from cats").Scan(&numCats)
	assert.Nil(t, err)
	check.Equal(t, 0, numCats)

	var numBlogPosts int
	err = db.QueryRowContext(ctx, "select count(*) from blog_posts").Scan(&numBlogPosts)
	assert.Nil(t, err)
	check.Equal(t, 0, numBlogPosts)
}
*/
