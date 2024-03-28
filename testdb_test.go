package pgtestdb_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	pgx "github.com/jackc/pgx/v5"      // "pgx" driver
	_ "github.com/jackc/pgx/v5/stdlib" // "pgx" driver
	_ "github.com/lib/pq"              // "postgres"
	"github.com/peterldowns/testy/assert"
	"github.com/peterldowns/testy/check"

	"github.com/peterldowns/pgtestdb"
	"github.com/peterldowns/pgtestdb/internal/sessionlock"
	"github.com/peterldowns/pgtestdb/migrators/common"
)

// You should wrap pgtestdb.New inside your own helper, like this,
// which sets up the db configuration (based on your own environment/configs)
// and passes an instance of the Migrator interface.
func New(t *testing.T) *sql.DB {
	t.Helper()
	dbconf := pgtestdb.Config{
		DriverName: "pgx",
		User:       "postgres",
		Password:   "password",
		Host:       "localhost",
		Port:       "5433",
		Options:    "sslmode=disable",
	}
	m := defaultMigrator()
	return pgtestdb.New(t, dbconf, m)
}

// Checks to make sure that the testdb is created succesfully and that all
// migrations are applied.
func TestNew(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := New(t)

	rows, err := db.QueryContext(ctx, "SELECT name FROM cats ORDER BY name ASC")
	assert.Nil(t, err)
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		assert.Nil(t, rows.Scan(&name))
		names = append(names, name)
	}
	check.Equal(t, []string{"daisy", "sunny"}, names)
}

func TestCustom(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dbconf := pgtestdb.Config{
		DriverName: "pgx",
		User:       "postgres",
		Password:   "password",
		Host:       "localhost",
		Port:       "5433",
		Options:    "sslmode=disable",
	}
	m := defaultMigrator()
	config := pgtestdb.Custom(t, dbconf, m)
	check.NotEqual(t, dbconf, *config)

	var conn *pgx.Conn
	var err error
	conn, err = pgx.Connect(ctx, config.URL())
	assert.Nil(t, err)
	defer func() {
		err := conn.Close(ctx)
		assert.Nil(t, err)
	}()

	var message string
	err = conn.QueryRow(ctx, "select 'hello world'").Scan(&message)
	assert.Nil(t, err)
}

// The Prepare() method of our dummy migrator should have enabled the `pg_trgm`
// and `pgcrypto` extensions. The `plpgsql` extension is always enabled by
// default. This test makes sure that these extensions are installed.
func TestExtensionsInstalled(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := New(t)
	rows, err := db.QueryContext(ctx, "SELECT extname FROM pg_extension ORDER BY extname ASC")
	assert.Nil(t, err)
	defer rows.Close()

	var extnames []string
	for rows.Next() {
		var extname string
		assert.Nil(t, rows.Scan(&extname))
		extnames = append(extnames, extname)
	}
	check.Equal(t, []string{"pg_trgm", "pgcrypto", "plpgsql"}, extnames)
}

// These two tests should show that creating many different testdbs in parallel
// is quite fast. Each of the tests creates and destroys 10 databases.
func TestParallel1(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		t.Run(fmt.Sprintf("subtest_%d", i), func(t *testing.T) {
			t.Parallel()
			db := New(t)

			var count int
			err := db.QueryRowContext(ctx, "SELECT COUNT(*) from cats").Scan(&count)
			assert.Nil(t, err)
			assert.Equal(t, 2, count)
		})
	}
}

// These two tests should show that creating many different testdbs in parallel
// is quite fast. Each of the tests creates and destroys 10 databases.
func TestParallel2(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		t.Run(fmt.Sprintf("subtest_%d", i), func(t *testing.T) {
			t.Parallel()
			db := New(t)

			var count int
			err := db.QueryRowContext(ctx, "SELECT COUNT(*) from cats").Scan(&count)
			assert.Nil(t, err)
			check.Equal(t, 2, count)
		})
	}
}

func TestDifferentHashesAlwaysResultInDifferentDatabases(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dbconf := pgtestdb.Config{
		DriverName: "pgx",
		User:       "postgres",
		Password:   "password",
		Host:       "localhost",
		Port:       "5433",
		Options:    "sslmode=disable",
	}
	// These two migrators have different hashes and they create databases with different schemas.
	// The xxx schema contains a table xxx, the yyy schema contains a table yyy.
	xxxm := &sqlMigrator{
		migrations: []string{
			"CREATE TABLE xxx (id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY)",
		},
	}
	yyym := &sqlMigrator{
		migrations: []string{
			"CREATE TABLE yyy (id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY)",
		},
	}
	// These two migrators should have different hashes.
	yyyh, err := yyym.Hash()
	assert.Nil(t, err)
	xxxh, err := xxxm.Hash()
	assert.Nil(t, err)
	assert.NotEqual(t, yyyh, xxxh)

	// Create two databases. They _should_ have different schemas.
	xxxdb := pgtestdb.New(t, dbconf, xxxm)
	yyydb := pgtestdb.New(t, dbconf, yyym)

	// But, the bug is that due to use of t.Once(), they will actually have the
	// same schema.  One of these two statements will always fail! Due to
	// ordering in this test, the xxx database gets created first, and the yyy
	// database will re-use that template (mistakenly!).
	//
	// In the case where we're writing a package and have multiple tests in
	// parallel, the order is dependent on whichever test runs first, which is
	// really annoying to debug.
	var countXXX int
	err = xxxdb.QueryRowContext(ctx, "select count(*) from xxx").Scan(&countXXX)
	if check.Nil(t, err) {
		check.Equal(t, 0, countXXX)
	}
	var countYYY int
	err = yyydb.QueryRowContext(ctx, "select count(*) from yyy").Scan(&countYYY)
	if check.Nil(t, err) {
		check.Equal(t, 0, countXXX)
	}
}

// This test confirms that due to testdb's locking strategy, even a migrator
// that uses advisory locks and runs a migration with "CREATE INDEX CONCURRENTLY"
// will succeed. pgtestdb will take an advisory lock on the primary database
// that it is connected to, NOT on the template database. This means that there
// is only ever one migrator running on the template database at a time, and there
// will never be any other migrators waiting or potentially contending an advisory
// lock on that database.
//
// Normally, if you have two connections to a database (c1 and c2), and they are
// contending an advisory lock, attempting to CREATE INDEX CONCCURENTLY will
// cause a deadlock error:
//
//	C1: SELECT pg_advisory_locK(1111) -- returns OK
//	C2: SELECT pg_advisory_locK(1111) -- hangs indefinitely, waiting on C1
//	C1: CREATE INDEX CONCURRENTLY ... -- fails with warning about deadlock, waiting on
//	                                  -- the C2 virtual transaction!
//
// Here, that's not an issue because template creation happens at most once, so
// C2, which is a second connection to the template database, will never exist.
func TestMigrationWithConcurrentCreate(t *testing.T) {
	t.Parallel()
	config := pgtestdb.Config{
		DriverName: "pgx",
		User:       "postgres",
		Password:   "password",
		Host:       "localhost",
		Port:       "5433",
		Options:    "sslmode=disable",
	}
	migrator := &sqlMigrator{
		migrations: []string{
			"CREATE TABLE users (id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY)",
			"CREATE INDEX CONCURRENTLY example_concurrent_index ON users (id)",
		},
	}
	for i := 0; i < 10; i++ {
		t.Run(fmt.Sprintf("subtest_%d", i), func(t *testing.T) {
			t.Parallel()
			_ = pgtestdb.New(t, config, migrator)
		})
	}
}

// pgtestdb.New should be able to connect with either lib/pq or pgx/stdlib.
func TestWithLibPqAndPgxStdlibDrivers(t *testing.T) {
	t.Parallel()
	baseConfig := pgtestdb.Config{
		User:     "postgres",
		Password: "password",
		Host:     "localhost",
		Port:     "5433",
		Options:  "sslmode=disable",
	}
	pgxConfig := baseConfig
	pgxConfig.DriverName = "pgx"
	pqConfig := baseConfig
	pqConfig.DriverName = "postgres"

	migrator := defaultMigrator()
	for i := 0; i < 10; i++ {
		t.Run(fmt.Sprintf("subtest_pgx_%d", i), func(t *testing.T) {
			t.Parallel()
			_ = pgtestdb.New(t, pgxConfig, migrator)
		})
	}
	for i := 0; i < 10; i++ {
		t.Run(fmt.Sprintf("subtest_pq_%d", i), func(t *testing.T) {
			t.Parallel()
			_ = pgtestdb.New(t, pqConfig, migrator)
		})
	}
}

// defaultMigrator is an implementation of the Migrator interface, and will
// create a `migrations` table and a `cats` table, with some data.
func defaultMigrator() pgtestdb.Migrator {
	return &sqlMigrator{
		preparations: []string{`
			CREATE EXTENSION pgcrypto;
			CREATE EXTENSION pg_trgm;
		`},
		migrations: []string{`
			-- as if this were a real migrations tool that kept track of migrations
			CREATE TABLE migrations (
				id TEXT PRIMARY KEY,
				applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
			);
			-- the "migration"
			CREATE TABLE cats (
				id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
				name text
			);
			INSERT INTO cats (name)
			VALUES ('daisy'), ('sunny');	
			-- recordkeeping
			INSERT INTO migrations (id)
			VALUES ('cats_0001');
		`},
		// These queries will fail if the tables don't exist.
		verifications: []string{
			"SELECT COUNT(*) from cats;",
			"SELECT COUNT(*) from migrations;",
		},
	}
}

// sqlMigrator is a test helper that satisfies the pgtestdb.Migrator interface.
type sqlMigrator struct {
	preparations  []string
	migrations    []string
	verifications []string
}

func (s *sqlMigrator) Hash() (string, error) {
	hash := common.NewRecursiveHash()
	for _, preparation := range s.preparations {
		hash.Add([]byte(preparation))
	}
	for _, migration := range s.migrations {
		hash.Add([]byte(migration))
	}
	return hash.String(), nil
}

func (s *sqlMigrator) Migrate(ctx context.Context, db *sql.DB, _ pgtestdb.Config) error {
	return sessionlock.With(ctx, db, "test-sql-migrator", func(conn *sql.Conn) error {
		for _, migration := range s.migrations {
			if _, err := conn.ExecContext(ctx, migration); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *sqlMigrator) Prepare(ctx context.Context, db *sql.DB, _ pgtestdb.Config) error {
	return sessionlock.With(ctx, db, "test-sql-migrator", func(conn *sql.Conn) error {
		for _, migration := range s.preparations {
			if _, err := conn.ExecContext(ctx, migration); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *sqlMigrator) Verify(ctx context.Context, db *sql.DB, _ pgtestdb.Config) error {
	for _, verification := range s.verifications {
		if _, err := db.ExecContext(ctx, verification); err != nil {
			return err
		}
	}
	return nil
}
