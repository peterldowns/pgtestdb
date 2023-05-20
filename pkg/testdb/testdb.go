package testdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
)

// TODO: should these be struct members, so that there can be multiple test databases used
// in the same package? That seems a little unnecessary... but probably good form?
var (
	tonce   sync.Once       //nolint:gochecknoglobals
	tconfig *templateConfig //nolint:gochecknoglobals
	terror  error           //nolint:gochecknoglobals
)

// templateConfig is the config for the created templateDB, which is basically
// connection string details and then its unique hash identifier. The hash is
// used when creating instances of the template, to make it easier to understand
// which template they came from.
type templateConfig struct {
	Config
	Hash string
}

// TODO: docs about wrapping this so that it's called with New(t) and nothing
// else, using a consistent config and migrationsDir
func New(t *testing.T, baseConfig Config, migrator Migrator) *sql.DB {
	ctx := context.Background()
	t.Helper()

	tplconf := ensureTemplateDB(t, baseConfig, migrator)

	baseDB, err := baseConfig.Connect()
	if err != nil {
		t.Fatalf("could not connect to database: %s", err)
		return nil // unreachable
	}
	t.Cleanup(func() {
		_ = baseDB.Close()
	})

	testConfig, err := createTestDBFromTemplateDB(baseDB, tplconf)
	if err != nil {
		t.Fatalf("failed to create testdb: %s", err)
		return nil // unreachable
	}

	testDB, err := testConfig.Connect()
	if err != nil {
		t.Fatalf("failed to connect to testdb: %s", err)
		return nil // unreachable
	}
	t.Cleanup(func() {
		// Close the testDB
		if err := testDB.Close(); err != nil {
			t.Fatalf("could not close test database: '%s': %s", testConfig.Database, err)
			return // uncreachable
		}
		// If the test failed, leave the testdb around for further investigation
		if t.Failed() {
			return
		}
		// Otherwise, remove the testdb from the server
		query := fmt.Sprintf("DROP DATABASE %s", testConfig.Database)
		if _, err := baseDB.ExecContext(ctx, query); err != nil {
			t.Fatalf("could not drop test database '%s': %s", testConfig.Database, err)
		}
	})

	return testDB
}

// ensureTemplateDB gets-or-creates a template database, and makes sure that it
// passes verification.
func ensureTemplateDB(
	t *testing.T,
	baseConf Config,
	migrator Migrator,
) templateConfig {
	tonce.Do(func() {
		hash, err := migrator.Hash()
		if err != nil {
			terror = fmt.Errorf("migrator failed to calculate hash: %w", err)
			return
		}
		templateConf := templateConfig{Config: baseConf, Hash: hash}
		templateConf.User = "testuser" // tODO: extract constant for this
		templateConf.Password = "testpassword"
		templateConf.Database = fmt.Sprintf("testdb_tpl_%s", hash)

		baseDB, err := baseConf.Connect()
		if err != nil {
			terror = fmt.Errorf("failed to connect to database: %w", err)
			return
		}
		defer baseDB.Close()
		ctx := context.Background()

		// Obtain a session-level advisory lock. The lock is released when the
		// session is closed.
		// TODO: why? already protected by Once? (in case another binary is
		// running in parallel?)
		// TDOO: use sessionlock.go helpers, base the lock ID on the template
		// database name.
		// TODO: also, explicitly unlock!
		query := "SELECT pg_advisory_lock(0)"
		if _, err := baseDB.ExecContext(ctx, query); err != nil {
			terror = fmt.Errorf("failed to acquire advisory lock: %w", err)
			return
		}

		var templateDBExists bool
		query = fmt.Sprintf(
			"SELECT EXISTS (SELECT FROM pg_database WHERE datname = '%s')",
			templateConf.Database,
		)
		if err := baseDB.QueryRowContext(ctx, query).Scan(&templateDBExists); err != nil {
			terror = fmt.Errorf("failed to check if templatedb already exists: %w", err)
			return
		}

		var templateDB *sql.DB
		if templateDBExists {
			templateDB, err = templateConf.Connect()
			if err != nil {
				terror = fmt.Errorf("failed to connect to templatedb %s: %w", templateConf.Database, err)
				return
			}
			defer templateDB.Close()
		} else {
			// Remove any existing template databases.
			// TODO: make this configurable behavior?
			query = "UPDATE pg_database SET datistemplate=false WHERE datname LIKE 'testdb_tpl_%'"
			if _, err := baseDB.ExecContext(ctx, query); err != nil {
				terror = fmt.Errorf("failed to mark all existing template dbs for deletion: %w", err)
				return
			}
			query = "SELECT datname FROM pg_database WHERE datname LIKE 'testdb_tpl_%' OR datname LIKE 'testdb_test_%'"
			rows, err := baseDB.QueryContext(ctx, query)
			if err != nil {
				terror = fmt.Errorf("failed to fetch database names for deletion: %w", err)
				return
			}
			defer rows.Close()
			for rows.Next() {
				var name string
				if err := rows.Scan(&name); err != nil {
					terror = fmt.Errorf("failed to read database name from row: %w", err)
					return
				}
				query = fmt.Sprintf("DROP DATABASE %s", name)
				if _, err := baseDB.ExecContext(ctx, query); err != nil {
					terror = fmt.Errorf("failed to drop database %s: %w", name, err)
					return
				}
			}

			// Create the test user/role with the appropriate permissions
			// TODO: allow configuring these permissions?
			var roleExists bool
			query = fmt.Sprintf(
				"SELECT EXISTS (SELECT from pg_catalog.pg_roles WHERE rolname = '%s')",
				templateConf.User,
			)
			if err := baseDB.QueryRowContext(ctx, query).Scan(&roleExists); err != nil {
				terror = fmt.Errorf("failed to detect if role %s exists: %w", templateConf.User, err)
				return
			}
			if !roleExists {
				query = fmt.Sprintf("CREATE ROLE %s", templateConf.User)
				if _, err := baseDB.ExecContext(ctx, query); err != nil {
					terror = fmt.Errorf("failed to create role %s: %w", templateConf.User, err)
					return
				}
				query = fmt.Sprintf(
					"ALTER ROLE %s WITH LOGIN PASSWORD '%s' NOSUPERUSER NOCREATEDB NOCREATEROLE",
					templateConf.User,
					templateConf.Password,
				)
				if _, err := baseDB.ExecContext(ctx, query); err != nil {
					terror = fmt.Errorf("failed to alter role and set password for %s: %w", templateConf.User, err)
					return
				}
			}

			// Create the templateDB
			// TODO: explore using ? and query params instead of %s.
			query = fmt.Sprintf("CREATE DATABASE %s OWNER %s", templateConf.Database, templateConf.User)
			if _, err := baseDB.ExecContext(ctx, query); err != nil {
				terror = fmt.Errorf("failed to create testdb %s: %w", templateConf.Database, err)
				return
			}

			// TODO: ability to run custom configuration code at this point in the
			// testdb setup?
			// - Fivetran role
			// - Extensions (trigram, pgcrypto)
			// - CREATEDB role
			templateDB, err = templateConf.Connect()
			if err != nil {
				terror = fmt.Errorf("failed to connect to templatedb %s: %w", templateConf.Database, err)
				return
			}
			defer templateDB.Close()

			// Grant privileges on the templateDB and all subsequently created objects
			// to the testDB user (who is also the templateDB user)
			for _, query = range []string{
				// TODO: is this necessary?
				fmt.Sprintf("GRANT ALL PRIVILEGES ON DATABASE %s TO %s", templateConf.Database, templateConf.User),
				// TODO: are these working?
				fmt.Sprintf("ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL PRIVILEGES ON TABLES TO %s", templateConf.User),
				fmt.Sprintf("ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL PRIVILEGES ON SEQUENCES TO %s", templateConf.User),
			} {
				if _, err := templateDB.ExecContext(ctx, query); err != nil {
					terror = fmt.Errorf(
						"failed to grant privileges on testdb %s to role %s: %w",
						templateConf.Database,
						templateConf.User,
						err,
					)
					return
				}
			}

			// Apply migrations one time, when creating the template database.
			// If this fails, keep the template database around so that the
			// developer can connect to it and investigate the failure (maybe
			// useful depending on the migration strategy).
			err = migrator.Migrate(ctx, templateDB)
			if err != nil {
				terror = fmt.Errorf("failed to migrated templatedb %s: %w", templateConf.Database, err)
				return
			}

			// TODO: add logging for each of these queries
			query = fmt.Sprintf("UPDATE pg_database SET datistemplate=true WHERE datname='%s'", templateConf.Database)
			if _, err := baseDB.ExecContext(ctx, query); err != nil {
				terror = fmt.Errorf("failed to mark testdb %s as template: %w", templateConf.Database, err)
				return
			}
		}

		// Even if we are re-using an existing template database, we will
		// attempt to verify that it was created successfully. This way if there
		// was ever a mistake or problem creating the template, a test will find
		// out at the site of `testdb.New` rather than later in the test due to
		// unexpected content in the database.
		//
		// We assume that verification is faster than performing the migrations,
		// and will not modify the template database.
		if err := migrator.Verify(ctx, templateDB); err != nil {
			terror = fmt.Errorf("failed to verify templatedb %s: %w", templateConf.Database, err)
			return
		}

		// If execution reaches this point, we have a template database that has
		// passed the verification step, so it should be able to be used in a
		// test.
		tconfig = &templateConf
		terror = nil
	})

	// If there were any errors creating the template database, log them and
	// immediately fail the test.
	if terror != nil {
		t.Errorf("failed to provision test database template: %s", terror)
	}
	if t.Failed() {
		t.FailNow()
	}
	return *tconfig
}

// createTestDBFromTemplateDB creates a new postgres database from a given
// template.
func createTestDBFromTemplateDB(baseDB *sql.DB, templateConf templateConfig) (*Config, error) {
	testConf := templateConf.Config
	testConf.Database = fmt.Sprintf(
		"testdb_test_%s_%s",
		templateConf.Hash,
		strings.ReplaceAll(uuid.New().String(), "-", ""),
	)

	// Create a test database from the template database. Using a template is
	// substantially faster than using pg_dump.
	ctx := context.Background()
	query := fmt.Sprintf(
		"CREATE DATABASE %s WITH TEMPLATE %s OWNER %s",
		testConf.Database,
		templateConf.Database,
		testConf.User,
	)
	if _, err := baseDB.ExecContext(ctx, query); err != nil {
		return nil, err
	}

	return &testConf, nil
}
