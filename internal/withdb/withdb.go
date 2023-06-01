// withdb is a simplified way of creating test databases, used to test the
// internal packages that testdb depends on.
package withdb

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"

	"github.com/peterldowns/testdb/internal/multierr"
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
func WithDB(ctx context.Context, driverName string, cb func(*sql.DB) error) (final error) {
	db, err := sql.Open(driverName, connectionString("postgres"))
	if err != nil {
		return fmt.Errorf("withdb(postgres) failed to open: %w", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			err = fmt.Errorf("withdb(postgres) failed to close: %w", err)
			final = multierr.Join(final, err)
		}
	}()

	testDBName, err := randomID("test_")
	if err != nil {
		return fmt.Errorf("withdb: random name failed: %w", err)
	}
	query := fmt.Sprintf("CREATE DATABASE %s", testDBName)
	if _, err := db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("withdb(%s) failed to create: %w", testDBName, err)
	}
	testDB, err := sql.Open("pgx", connectionString(testDBName))
	if err != nil {
		return fmt.Errorf("withdb(%s) failed to open: %w", testDBName, err)
	}
	defer func() {
		if err := testDB.Close(); err != nil {
			err = fmt.Errorf("withdb(%s) failed to close: %w", testDBName, err)
			final = multierr.Join(final, err)
		}
		query := fmt.Sprintf("DROP DATABASE %s", testDBName)
		if _, err = db.ExecContext(ctx, query); err != nil {
			err = fmt.Errorf("withdb(%s) failed to drop: %w", testDBName, err)
			final = multierr.Join(final, err)
		}
	}()
	return cb(testDB)
}

func randomID(prefix string) (string, error) {
	bytes := make([]byte, 32)
	hash := md5.New()
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(hash.Sum(bytes))), nil
}

func connectionString(dbname string) string {
	return fmt.Sprintf("postgres://postgres:password@localhost:5433/%s?sslmode=disable", dbname)
}
