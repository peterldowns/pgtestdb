package dbmatemigrator_test

import (
	"context"
	"embed"
	"testing"

	"github.com/peterldowns/testy/assert"
	"github.com/peterldowns/testy/check"

	"github.com/peterldowns/testdb"
	"github.com/peterldowns/testdb/migrators/dbmatemigrator"
)

func TestDbmateMigratorWithOptions(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	m := dbmatemigrator.New(
		dbmatemigrator.WithDir("./migrations", "more"),
		dbmatemigrator.WithTableName("dbmate_migrations_example"),
	)
	db := testdb.New(t, testdb.Config{
		Host:     "localhost",
		User:     "postgres",
		Password: "password",
		Port:     "5433",
		Options:  "sslmode=disable",
	}, m)
	assert.NotEqual(t, nil, db)

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

	var numMigrations int
	err = db.QueryRowContext(ctx, "select count(*) from dbmate_migrations_example;").Scan(&numMigrations)
	assert.Nil(t, err)
	check.Equal(t, 3, numMigrations)

	var funcResult string
	err = db.QueryRowContext(ctx, "select testdb();").Scan(&funcResult)
	assert.Nil(t, err)
	check.Equal(t, "dummy", funcResult)
}

//go:embed migrations/*.sql more/*.sql
var migrationsFS embed.FS

func TestDbmateMigratorWithFS(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	m := dbmatemigrator.New(
		dbmatemigrator.WithFS(migrationsFS),
		dbmatemigrator.WithDir("migrations", "more"),
		dbmatemigrator.WithTableName("dbmate_migrations_example"),
	)
	db := testdb.New(t, testdb.Config{
		Host:     "localhost",
		User:     "postgres",
		Password: "password",
		Port:     "5433",
		Options:  "sslmode=disable",
	}, m)
	assert.NotEqual(t, nil, db)

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

	var numMigrations int
	err = db.QueryRowContext(ctx, "select count(*) from dbmate_migrations_example;").Scan(&numMigrations)
	assert.Nil(t, err)
	check.Equal(t, 3, numMigrations)

	var funcResult string
	err = db.QueryRowContext(ctx, "select testdb();").Scan(&funcResult)
	assert.Nil(t, err)
	check.Equal(t, "dummy", funcResult)
}
