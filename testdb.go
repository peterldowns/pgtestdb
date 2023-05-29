package testdb

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/peterldowns/testdb/internal/safeonce"
	"github.com/peterldowns/testdb/internal/sessionlock"
)

const (
	// TestUser is the username for connecting to each test database.
	TestUser = "testdbuser"
	// TestPassword is the password for connecting to each test database.
	TestPassword = "password"
)

// TODO: docs about wrapping this so that it's called with New(t) and nothing
// else, using a consistent config and migrationsDir
func New(t *testing.T, conf Config, migrator Migrator) *sql.DB {
	t.Helper()
	ctx := context.Background()
	baseDB, err := conf.connect()
	if err != nil {
		// TODO: optionally, allow for Functional Options to `t.Skip()`
		// instead of `t.Fatal()` so that when the database is down, tests
		// are ignored instead of failed?
		t.Fatalf("could not connect to database: %s", err)
		return nil // unreachable
	}

	if err := ensureUser(ctx, baseDB); err != nil {
		t.Fatalf("could not create testdb user: %s", err)
	}

	template, err := getOrCreateTemplate(ctx, baseDB, conf, migrator)
	if err != nil {
		t.Fatal(err)
	}

	instance, err := createInstance(ctx, baseDB, *template)
	if err != nil {
		t.Fatalf("failed to create testdb: %s", err)
		return nil // unreachable
	}
	t.Logf("testdbconf: %s", instance.URL())

	db, err := instance.connect()
	if err != nil {
		t.Fatalf("failed to connect to testdb: %s", err)
		return nil // unreachable
	}
	t.Cleanup(func() {
		// Close the testDB
		if err := db.Close(); err != nil {
			t.Fatalf("could not close test database: '%s': %s", instance.Database, err)
			return // uncreachable
		}
		// If the test failed, leave the testdb around for further investigation
		if t.Failed() {
			return
		}
		// Otherwise, remove the testdb from the server
		query := fmt.Sprintf("DROP DATABASE %s", instance.Database)
		if _, err := baseDB.ExecContext(ctx, query); err != nil {
			t.Fatalf("could not drop test database '%s': %s", instance.Database, err)
		}
	})

	// Even if we are re-using an existing template database, we will
	// attempt to verify that it was created successfully. This way if there
	// was ever a mistake or problem creating the template, a test will find
	// out at the site of `testdb.New` rather than later in the test due to
	// unexpected content in the database.
	//
	// We assume that verification is >>> faster than performing the migrations,
	// and is therefore safe to run at the beginning of each test.
	if err := migrator.Verify(ctx, db, *instance); err != nil {
		t.Fatal(fmt.Errorf("test database failed verification %s: %w", instance.Database, err))
	}

	return db
}

// userInit is used to guarantee that we only get-or-create the testdb user
// at most once per program.
var userInit safeonce.Var[any] = safeonce.NewVar[any]() //nolint:gochecknoglobals

func ensureUser(
	ctx context.Context,
	baseDB *sql.DB,
) error {
	_, err := userInit.Set(func() (*any, error) {
		return nil, sessionlock.With(ctx, baseDB, "testdb-user", func() error {
			// Get-or-create a role/user dedicated to connecting to these test databases.
			var roleExists bool
			query := "SELECT EXISTS (SELECT from pg_catalog.pg_roles WHERE rolname = $1)"
			if err := baseDB.QueryRowContext(ctx, query, TestUser).Scan(&roleExists); err != nil {
				return fmt.Errorf("failed to detect if role %s exists: %w", TestUser, err)
			}
			if roleExists {
				return nil
			}
			if !roleExists {
				query = fmt.Sprintf(`CREATE ROLE "%s"`, TestUser)
				if _, err := baseDB.ExecContext(ctx, query); err != nil {
					return fmt.Errorf("failed to create role %s: %w", TestUser, err)
				}
				query = fmt.Sprintf(
					`ALTER ROLE "%s" WITH LOGIN PASSWORD '%s' NOSUPERUSER NOCREATEDB NOCREATEROLE`,
					TestUser,
					TestPassword,
				)
				if _, err := baseDB.ExecContext(ctx, query); err != nil {
					return fmt.Errorf("failed to alter role and set password for %s: %w", TestUser, err)
				}
			}
			return nil
		})
	})
	return err
}

// templateState keeps the state of a single template database, so that each
// program only attempts to create/migrate the template database at most once.
type templateState struct {
	conf Config
	hash string
}

var states safeonce.Map[string, templateState] = safeonce.NewMap[string, templateState]() //nolint:gochecknoglobals

// getOrCreateTemplate will get-or-create a template database, synchronizing at the
// golang level (with the stateLocks map, so that each template database is
// get-or-created at most once) and at the postgres level (with advisory locks,
// so that there are no conflicts between multiple programs trying to create
// the template.)
//
// If there was a database error during template creation, the program that
// attempted the creation will set state.error, so subsequent attempts to access
// the template from within the same golang program will just return that error.
//
// This means that:
// - migrations are only run once per template database per golang program / package under test.
// - you don't need to manually clear out "broken" templates between test suite runs.
func getOrCreateTemplate(
	ctx context.Context,
	baseDB *sql.DB,
	dbconf Config,
	migrator Migrator,
) (*templateState, error) {
	// Get the unique hash of the database.
	hash, err := migrator.Hash()
	if err != nil {
		return nil, err
	}
	return states.Set(hash, func() (*templateState, error) {
		// This function runs once per program, but only synchronizes access
		// within a single program. When running larger test suites, each
		// package's tests may run in parallel, which means this does not
		// perfectly synchronize interaction with the database.
		state := templateState{}
		state.conf = dbconf
		// Use the TestUser/TestPassword because we guaranteed it to exist
		// earlier.
		state.conf.User = TestUser
		state.conf.Password = TestPassword
		state.conf.Database = fmt.Sprintf("testdb_tpl_%s", hash)
		// We synchronize the creation of the template database across programs
		// by using a Postgres advisory lock.
		err := sessionlock.With(ctx, baseDB, "testdb-"+hash, func() error {
			return ensureTemplate(ctx, baseDB, migrator, &state)
		})
		if err != nil {
			return nil, err
		}
		return &state, nil
	})
}

// ensureTemplate does what it says. It uses the 'datistemplate' column
// to mark a template as having been successfully created, and does not set
// 'datistemplate = true' until after the database has been created and
// migrated. If it finds a template database where 'datistemplate = false', it
// drops and then attempts to recreate that database.
func ensureTemplate(
	ctx context.Context,
	baseDB *sql.DB,
	migrator Migrator,
	state *templateState,
) error {
	// If the templateDB already exists, and is marked as a template, it means we
	// migrated successfully and we can exit right now.
	var templateDBExists bool
	query := "SELECT EXISTS (SELECT FROM pg_database WHERE datname = $1 AND datistemplate = true)"
	if err := baseDB.QueryRowContext(ctx, query, state.conf.Database).Scan(&templateDBExists); err != nil {
		return fmt.Errorf("failed to check if templatedb already exists: %w", err)
	}
	if templateDBExists {
		return nil
	}

	// If it exists and isn't marked as a template, we failed to migrate it properly, so we
	// should remove it entirely and then recreate it.
	query = fmt.Sprintf(`DROP DATABASE IF EXISTS "%s"`, state.conf.Database)
	if _, err := baseDB.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("failed to drop broken template database %s: %w", state.conf.Database, err)
	}

	query = fmt.Sprintf(`CREATE DATABASE "%s" OWNER "%s"`, state.conf.Database, state.conf.User)
	if _, err := baseDB.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("failed to create testdb %s: %w", state.conf.Database, err)
	}

	// Grant all privileges on the template to the test user.
	for _, query = range []string{
		fmt.Sprintf(`GRANT ALL PRIVILEGES ON DATABASE "%s" TO "%s"`, state.conf.Database, state.conf.User),
		fmt.Sprintf(`ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL PRIVILEGES ON TABLES TO "%s"`, state.conf.User),
		fmt.Sprintf(`ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL PRIVILEGES ON SEQUENCES TO "%s"`, state.conf.User),
	} {
		if _, err := baseDB.ExecContext(ctx, query); err != nil {
			return fmt.Errorf(
				"failed to grant privileges on testdb %s to role %s: %w",
				state.conf.Database,
				state.conf.User,
				err,
			)
		}
	}

	// Connect to the template.
	templateDB, err := state.conf.connect()
	if err != nil {
		return fmt.Errorf("failed to connect to templatedb %s: %w", state.conf.Database, err)
	}
	defer templateDB.Close()

	// Apply the Migrator logic one time, when creating the template database.
	// If this fails, the template database will remain and the developer can
	// connect to it and investigate the failure. Subsequent attempts to create
	// the template will remove it, since it didn't get marked as complete
	// (datistemplate=true).
	if err := migrator.Prepare(ctx, templateDB, state.conf); err != nil {
		return fmt.Errorf("failed to prepare %s: %w", state.conf.Database, err)
	}
	if err := migrator.Migrate(ctx, templateDB, state.conf); err != nil {
		return fmt.Errorf("failed to migrate %s: %w", state.conf.Database, err)
	}

	// Finalize the creation of the template by marking it as a
	// template.
	query = "UPDATE pg_database SET datistemplate = true WHERE datname=$1"
	if _, err := baseDB.ExecContext(ctx, query, state.conf.Database); err != nil {
		return fmt.Errorf("failed to mark testdb %s as template: %w", state.conf.Database, err)
	}
	return nil
}

// createInstance creates a new test database instance by cloning a template.
func createInstance(
	ctx context.Context,
	baseDB *sql.DB,
	template templateState,
) (*Config, error) {
	testConf := template.conf
	testConf.Database = fmt.Sprintf(
		"testdb_tpl_%s_inst_%s",
		template.hash,
		randomID(),
	)
	query := fmt.Sprintf(
		`CREATE DATABASE "%s" WITH TEMPLATE "%s" OWNER "%s"`,
		testConf.Database,
		template.conf.Database,
		testConf.User,
	)
	if _, err := baseDB.ExecContext(ctx, query); err != nil {
		return nil, err
	}
	return &testConf, nil
}

// randomID is a helper for coming up with the names of the instance databases.
// It uses 32 random bits in the name, which means collisions are unlikely.
func randomID() string {
	bytes := make([]byte, 4)
	hash := md5.New()
	_, err := rand.Read(bytes)
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(hash.Sum(bytes))
}
