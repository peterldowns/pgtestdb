package golangmigrator_test

import (
	"context"
	"embed"
	"testing"

	"github.com/peterldowns/testy/assert"
	"github.com/peterldowns/testy/check"

	"github.com/peterldowns/testdb"
	"github.com/peterldowns/testdb/migrators/golangmigrator"
)

//go:embed migrations/*.sql
var exampleFS embed.FS

func TestMigrateFromEmbeddedFS(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
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

	// Make sure we ran both migrations.
	assert.NoFailures(t, func() {
		var version int
		err := db.QueryRowContext(ctx, "select version from schema_migrations;").Scan(&version)
		assert.Nil(t, err)
		check.Equal(t, 2, version)

		var dirty bool
		err = db.QueryRowContext(ctx, "select dirty from schema_migrations;").Scan(&dirty)
		assert.Nil(t, err)
		check.False(t, dirty)
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

func TestMigrateFromDisk(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	gm := golangmigrator.New("migrations")
	db := testdb.New(t, testdb.Config{
		Host:     "localhost",
		User:     "postgres",
		Password: "password",
		Port:     "5433",
		Options:  "sslmode=disable",
	}, gm)
	assert.NotEqual(t, nil, db)

	// Make sure we ran both migrations.
	assert.NoFailures(t, func() {
		var version int
		err := db.QueryRowContext(ctx, "select version from schema_migrations;").Scan(&version)
		assert.Nil(t, err)
		check.Equal(t, 2, version)

		var dirty bool
		err = db.QueryRowContext(ctx, "select dirty from schema_migrations;").Scan(&dirty)
		assert.Nil(t, err)
		check.False(t, dirty)
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
