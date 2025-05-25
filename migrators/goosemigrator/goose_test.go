package goosemigrator_test

import (
	"context"
	"embed"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib" // "pgx" driver
	"github.com/peterldowns/testy/assert"
	"github.com/peterldowns/testy/check"

	"github.com/peterldowns/pgtestdb"
	"github.com/peterldowns/pgtestdb/migrators/goosemigrator"
)

func TestGooseMigratorFromDiskWithSubdirectory(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	m := goosemigrator.New("./migrations")
	db := pgtestdb.New(t, pgtestdb.Config{
		DriverName: "pgx",
		Host:       "localhost",
		User:       "postgres",
		Password:   "password",
		Port:       "5433",
		Options:    "sslmode=disable",
	}, m)
	assert.NotEqual(t, nil, db)

	assert.NoFailures(t, func() {
		var lastAppliedMigration int
		err := db.QueryRowContext(ctx, "select max(version_id) from goose_db_version").Scan(&lastAppliedMigration)
		assert.Nil(t, err)
		check.Equal(t, 2, lastAppliedMigration)
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

func TestGooseMigratorFromDiskWithParentDirectoryRelativePath(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	m := goosemigrator.New("../../migrators/goosemigrator/migrations")
	db := pgtestdb.New(t, pgtestdb.Config{
		DriverName: "pgx",
		Host:       "localhost",
		User:       "postgres",
		Password:   "password",
		Port:       "5433",
		Options:    "sslmode=disable",
	}, m)
	assert.NotEqual(t, nil, db)

	assert.NoFailures(t, func() {
		var lastAppliedMigration int
		err := db.QueryRowContext(ctx, "select max(version_id) from goose_db_version").Scan(&lastAppliedMigration)
		assert.Nil(t, err)
		check.Equal(t, 2, lastAppliedMigration)
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

func TestGooseMigratorFromEmbedFS(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

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

	assert.NoFailures(t, func() {
		var lastAppliedMigration int
		err := db.QueryRowContext(ctx, "select max(version_id) from goose_example_migrations").Scan(&lastAppliedMigration)
		assert.Nil(t, err)
		check.Equal(t, 2, lastAppliedMigration)
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
