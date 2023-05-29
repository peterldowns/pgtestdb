package atlasmigrator_test

import (
	"context"
	"testing"

	"github.com/peterldowns/testy/assert"
	"github.com/peterldowns/testy/check"

	"github.com/peterldowns/testdb"
	"github.com/peterldowns/testdb/migrators/atlasmigrator"
)

func TestDirMigrator(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	m := atlasmigrator.NewDirMigrator("migrations")
	db := testdb.New(t, testdb.Config{
		Host:     "localhost",
		User:     "postgres",
		Password: "password",
		Port:     "5433",
		Options:  "sslmode=disable",
	}, m)
	assert.NotEqual(t, nil, db)

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
