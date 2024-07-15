package bunmigrator_test

import (
	"context"
	"embed"
	"testing"

	"github.com/peterldowns/testy/assert"
	"github.com/peterldowns/testy/check"
	_ "github.com/uptrace/bun/driver/pgdriver"

	"github.com/peterldowns/pgtestdb"
	"github.com/peterldowns/pgtestdb/migrators/bunmigrator"
)

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

	assert.NoFailures(t, func() {
		var numMigrationsRan int
		err := db.QueryRowContext(ctx, "select count(*) from bun_migrations").Scan(&numMigrationsRan)
		assert.Nil(t, err)
		check.Equal(t, 2, numMigrationsRan)
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

	assert.NoFailures(t, func() {
		var numMigrationsRan int
		err := db.QueryRowContext(ctx, "select count(*) from bun_migrations").Scan(&numMigrationsRan)
		assert.Nil(t, err)
		check.Equal(t, 2, numMigrationsRan)
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
