package testdb_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	_ "github.com/lib/pq"

	"github.com/peterldowns/testdb/pkg/testdb"

	"github.com/peterldowns/testy/assert"
	"github.com/peterldowns/testy/check"
)

// migrator is an implementation of the Migrator interface, and will
// create a `migrations` table and a `cats` table, with some data.
type migrator struct{}

func (*migrator) Hash() (string, error) {
	return "dummyhash", nil
}

func (*migrator) Migrate(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
-- as if this were a real migrations tool that kept track of migrations
CREATE TABLE migrations (
	id TEXT PRIMARY KEY,
	applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- the "migration" that we apply
CREATE TABLE cats (
	id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	name text
);
INSERT INTO cats (name)
VALUES ('daisy'), ('sunny');	

-- recordkeeping
INSERT INTO migrations (id)
VALUES ('cats_0001');
`)
	if err != nil {
		return err
	}
	return nil
}

func (*migrator) Verify(ctx context.Context, db *sql.DB) error {
	rows, err := db.QueryContext(ctx, "SELECT id FROM migrations ORDER BY id ASC")
	if err != nil {
		return err
	}
	var migrations []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return err
		}
		migrations = append(migrations, id)
	}
	if !(len(migrations) == 1 && migrations[0] == "cats_0001") {
		return fmt.Errorf("the migrations failed to apply")
	}
	return nil
}

// We expect that you will wrap testdb.New inside your own helper, like this,
// which sets up the db configuration (based on your own environment/configs)
// and passes an instance of the Migrator interface.
func new(t *testing.T) *sql.DB {
	dbconf := testdb.Config{
		User:     "postgres",
		Password: "password",
		Host:     "localhost",
		Port:     "5433",
		Database: "postgres",
		Options:  "sslmode=disable",
	}
	m := &migrator{}
	return testdb.New(t, dbconf, m)
}

func TestNew(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := new(t)

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

// These two tests should show that creating many different testdbs in parallel
// is quite fast. Each of the tests creates and destroys 10 databases.
func TestParallel1(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		t.Run(fmt.Sprintf("subtest_%d", i), func(t *testing.T) {
			t.Parallel()
			db := new(t)

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
			db := new(t)

			var count int
			err := db.QueryRowContext(ctx, "SELECT COUNT(*) from cats").Scan(&count)
			assert.Nil(t, err)
			check.Equal(t, 2, count)
		})
	}
}
