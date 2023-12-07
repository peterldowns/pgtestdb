// This file contains all of the examples from README.md
package pgtestdb_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/jackc/pgx/v5"          // registers the "pgx" driver
	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" driver
	_ "github.com/lib/pq"              // registers the "postgres" driver
	"github.com/peterldowns/testy/assert"
	"github.com/peterldowns/testy/check"

	"github.com/peterldowns/pgtestdb"
)

func TestMyExample(t *testing.T) {
	// pgtestdb is concurrency safe, go crazy, run a lot of tests at once
	t.Parallel()
	// You should connect as an admin user. Use a dedicated server explicitly
	// for tests, do NOT use your production database.
	conf := pgtestdb.Config{
		DriverName: "pgx",
		User:       "postgres",
		Password:   "password",
		Host:       "localhost",
		Port:       "5433",
		Options:    "sslmode=disable",
	}
	// You'll want to use a real migrator, this is just an example. See
	// the rest of the docs for more information.
	var migrator pgtestdb.Migrator = pgtestdb.NoopMigrator{}
	db := pgtestdb.New(t, conf, migrator)
	// If there is any sort of error, the test will have ended with t.Fatal().
	// No need to check errors! Go ahead and use the database.
	var message string
	err := db.QueryRow("select 'hello world'").Scan(&message)
	assert.Nil(t, err)
	assert.Equal(t, "hello world", message)
}

// NewDB is a helper that returns an open connection to a unique and isolated
// test database, fully migrated and ready for you to query.
func NewDB(tb testing.TB) *sql.DB {
	tb.Helper()
	conf := pgtestdb.Config{
		DriverName: "pgx",
		User:       "postgres",
		Password:   "password",
		Host:       "localhost",
		Port:       "5433",
		Options:    "sslmode=disable",
	}
	// You'll want to use a real migrator, this is just an example. See the rest
	// of the docs for more information.
	var migrator pgtestdb.Migrator = pgtestdb.NoopMigrator{}
	return pgtestdb.New(tb, conf, migrator)
}

func TestAQuery(t *testing.T) {
	t.Parallel()
	db := NewDB(t) // this is the helper defined above

	var result string
	err := db.QueryRow("SELECT 'hello world'").Scan(&result)
	check.Nil(t, err)
	check.Equal(t, "hello world", result)
}

// NewPgx is a helper that returns an open pgx connection to a unique and
// isolated test database, fully migrated and ready for you to query.
func NewPgx(tb testing.TB, ctx context.Context) *pgx.Conn {
	tb.Helper()
	conf := pgtestdb.Config{
		DriverName: "pgx",
		User:       "postgres",
		Password:   "password",
		Host:       "localhost",
		Port:       "5433",
		Options:    "sslmode=disable",
	}
	// You'll want to use a real migrator, this is just an example. See the rest
	// of the docs for more information.
	var migrator pgtestdb.Migrator = pgtestdb.NoopMigrator{}
	instance := pgtestdb.NewInstance(tb, conf, migrator)
	conn, err := pgx.Connect(ctx, instance.URL())
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() {
		conn.Close(ctx)
	})
	return conn
}

// TestWithNewPgx uses the NewPgx helper to get a pgx connection to a test.
func TestWithNewPgx(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	conn := NewPgx(t, ctx)
	rows, _ := conn.Query(ctx, "select 'hello world'")
	message, err := pgx.CollectOneRow(rows, pgx.RowTo[string])
	assert.Nil(t, err)
	assert.Equal(t, "hello world", message)
}

func TestWithLibPqDriver(t *testing.T) {
	t.Parallel()
	pqConf := pgtestdb.Config{
		DriverName: "postgres", // uses the lib/pq driver
		User:       "postgres",
		Password:   "password",
		Host:       "localhost",
		Port:       "5433",
		Options:    "sslmode=disable",
	}
	migrator := pgtestdb.NoopMigrator{}
	db := pgtestdb.New(t, pqConf, migrator)

	var message string
	err := db.QueryRow("select 'hello world'").Scan(&message)
	assert.Nil(t, err)
	assert.Equal(t, "hello world", message)
}

func TestWithPgxStdlibDriver(t *testing.T) {
	t.Parallel()
	pgxConf := pgtestdb.Config{
		DriverName: "pgx", // uses the pgx/stdlib driver
		User:       "postgres",
		Password:   "password",
		Host:       "localhost",
		Port:       "5433",
		Options:    "sslmode=disable",
	}
	migrator := pgtestdb.NoopMigrator{}
	db := pgtestdb.New(t, pgxConf, migrator)

	var message string
	err := db.QueryRow("select 'hello world'").Scan(&message)
	assert.Nil(t, err)
	assert.Equal(t, "hello world", message)
}
