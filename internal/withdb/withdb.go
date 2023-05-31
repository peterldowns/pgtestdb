package withdb

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib" // pgx driver for postgres
)

// WithDB is a helper for writing postgres-backed tests. It will:
// - connect to a local postgres server (see docker-compose.yml)
// - create a new, empty test database with a unique name
// - open a connection to that test database
// - run the `cb` function
// - remove the test database
//
// This is designed to be an internal helper for testing other database-related
// packages, and should not be relied upon externally.
//
// For a package that will automatically create a database with migrations
// applied, check out `pkg/testdb`.
func WithDB(ctx context.Context, cb func(*sql.DB)) error {
	db, err := sql.Open("pgx", connectionString("postgres"))
	if err != nil {
		return err
	}
	defer db.Close()

	testDBName := randomID("test_")
	query := fmt.Sprintf("CREATE DATABASE %s", testDBName)
	if _, err := db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("could not create new database template: %w", err)
	}
	testDB, err := sql.Open("pgx", connectionString(testDBName))
	if err != nil {
		return err
	}
	defer func() {
		if err := testDB.Close(); err != nil {
			panic(err)
		}
		query := fmt.Sprintf("DROP DATABASE %s", testDBName)
		if _, err = db.ExecContext(ctx, query); err != nil {
			panic(err)
		}
	}()

	cb(testDB)
	return nil
}

func randomID(prefix string) string {
	bytes := make([]byte, 32)
	hash := md5.New()
	_, err := rand.Read(bytes)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(hash.Sum(bytes)))
}

func connectionString(dbname string) string {
	return fmt.Sprintf("postgres://postgres:password@localhost:5433/%s?sslmode=disable", dbname)
}