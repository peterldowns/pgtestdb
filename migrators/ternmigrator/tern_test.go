package ternmigrator_test

import (
	"embed"
	"fmt"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib" // "pgx" driver
	"github.com/peterldowns/testy/assert"
	"github.com/peterldowns/testy/check"
	"golang.org/x/net/context"

	"github.com/peterldowns/pgtestdb"
	"github.com/peterldowns/pgtestdb/migrators/ternmigrator"
)

func TestTernMigratorFromDisk(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	m := ternmigrator.New("migrations")
	db := pgtestdb.New(t, pgtestdb.Config{
		DriverName: "pgx",
		Host:       "localhost",
		User:       "postgres",
		Password:   "password",
		Port:       "5433",
		Options:    "sslmode=disable",
	}, m)

	assert.NoFailures(t, func() {
		var numMigrationsRan int
		query := fmt.Sprintf("SELECT MAX(version) FROM %s;", m.TableName)
		assert.Nil(t, db.QueryRowContext(ctx, query).Scan(&numMigrationsRan))
		check.Equal(t, 2, numMigrationsRan)
	})

	var numUsers int
	err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&numUsers)
	assert.Nil(t, err)
	check.Equal(t, 0, numUsers)

	var numCats int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM cats").Scan(&numCats)
	assert.Nil(t, err)
	check.Equal(t, 0, numCats)

	var numBlogPosts int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM blog_posts").Scan(&numBlogPosts)
	assert.Nil(t, err)
	check.Equal(t, 0, numBlogPosts)
}

//go:embed migrations/*.sql
var exampleFS embed.FS

func TestTernMigratorFromFS(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	m := ternmigrator.New("migrations", ternmigrator.WithFS(exampleFS))
	db := pgtestdb.New(t, pgtestdb.Config{
		DriverName: "pgx",
		Host:       "localhost",
		User:       "postgres",
		Password:   "password",
		Port:       "5433",
		Options:    "sslmode=disable",
	}, m)

	assert.NoFailures(t, func() {
		var numMigrationsRan int
		query := fmt.Sprintf("SELECT MAX(version) FROM %s;", m.TableName)
		assert.Nil(t, db.QueryRowContext(ctx, query).Scan(&numMigrationsRan))
		check.Equal(t, 2, numMigrationsRan)
	})

	var numUsers int
	err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&numUsers)
	assert.Nil(t, err)
	check.Equal(t, 0, numUsers)

	var numCats int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM cats").Scan(&numCats)
	assert.Nil(t, err)
	check.Equal(t, 0, numCats)

	var numBlogPosts int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM blog_posts").Scan(&numBlogPosts)
	assert.Nil(t, err)
	check.Equal(t, 0, numBlogPosts)
}

func TestTernMigratorWithTableName(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	m := ternmigrator.New("migrations", ternmigrator.WithTableName("some_other_table"))
	db := pgtestdb.New(t, pgtestdb.Config{
		DriverName: "pgx",
		Host:       "localhost",
		User:       "postgres",
		Password:   "password",
		Port:       "5433",
		Options:    "sslmode=disable",
	}, m)

	assert.NoFailures(t, func() {
		var numMigrationsRan int
		query := fmt.Sprintf("SELECT MAX(version) FROM %s;", m.TableName)
		assert.Nil(t, db.QueryRowContext(ctx, query).Scan(&numMigrationsRan))
		check.Equal(t, 2, numMigrationsRan)
	})

	var numUsers int
	err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&numUsers)
	assert.Nil(t, err)
	check.Equal(t, 0, numUsers)

	var numCats int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM cats").Scan(&numCats)
	assert.Nil(t, err)
	check.Equal(t, 0, numCats)

	var numBlogPosts int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM blog_posts").Scan(&numBlogPosts)
	assert.Nil(t, err)
	check.Equal(t, 0, numBlogPosts)
}
