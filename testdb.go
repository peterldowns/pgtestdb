package testdb

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"sync"
	"testing"

	"github.com/peterldowns/testdb/internal/sessionlock"
)

var (
	tonce   sync.Once       //nolint:gochecknoglobals
	tconfig *templateConfig //nolint:gochecknoglobals
	terror  error           //nolint:gochecknoglobals
)

// templateConfig is used only internally, and contains the config for the
// created templateDB, which is basically connection string details and then its
// unique hash identifier. The hash is used when creating instances of the
// template, to make it easier to understand which template they came from.
type templateConfig struct {
	Config
	Hash string
}

// TODO: docs about wrapping this so that it's called with New(t) and nothing
// else, using a consistent config and migrationsDir
func New(t *testing.T, conf Config, migrator Migrator) *sql.DB {
	t.Helper()
	ctx := context.Background()

	templateConf := ensureTemplateDB(t, conf, migrator)

	baseDB, err := conf.connect()
	if err != nil {
		// TODO: optionally, allow for Functional Options to `t.Skip()`
		// instead of `t.Fatal()` so that when the database is down, tests
		// are ignored instead of failed?
		t.Fatalf("could not connect to database: %s", err)
		return nil // unreachable
	}
	t.Cleanup(func() {
		_ = baseDB.Close()
	})

	instanceConf, err := createTestDBFromTemplateDB(baseDB, templateConf)
	if err != nil {
		t.Fatalf("failed to create testdb: %s", err)
		return nil // unreachable
	}

	t.Logf("testdbconf: %s", instanceConf.URL())
	instanceDB, err := instanceConf.connect()
	if err != nil {
		t.Fatalf("failed to connect to testdb: %s", err)
		return nil // unreachable
	}
	t.Cleanup(func() {
		// Close the testDB
		if err := instanceDB.Close(); err != nil {
			t.Fatalf("could not close test database: '%s': %s", instanceConf.Database, err)
			return // uncreachable
		}
		// If the test failed, leave the testdb around for further investigation
		if t.Failed() {
			return
		}
		// Otherwise, remove the testdb from the server
		query := fmt.Sprintf("DROP DATABASE %s", instanceConf.Database)
		if _, err := baseDB.ExecContext(ctx, query); err != nil {
			t.Fatalf("could not drop test database '%s': %s", instanceConf.Database, err)
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
	if err := migrator.Verify(ctx, instanceDB, *instanceConf); err != nil {
		t.Fatal(fmt.Errorf("test database failed verification %s: %w", instanceConf.Database, err))
	}
	return instanceDB
}

// ensureTemplateDB gets-or-creates a template database, and makes sure that it
// passes verification.
func ensureTemplateDB(
	t *testing.T,
	dbconf Config,
	migrator Migrator,
) templateConfig {
	t.Helper()
	// This setup step will run once per test binary execution. If
	// a test database does not yet exist, it will be created.
	tonce.Do(func() {
		hash, err := migrator.Hash()
		if err != nil {
			terror = fmt.Errorf("migrator.Hash() failed: %w", err)
			return
		}
		templateConf := templateConfig{Config: dbconf, Hash: hash}
		// TODO: extract option/config for the user and password to use for
		// the template/test databases.
		templateConf.User = "testuser"
		templateConf.Password = "testpassword"
		templateConf.Database = fmt.Sprintf("testdb_tpl_%s", hash)

		baseDB, err := dbconf.connect()
		if err != nil {
			terror = fmt.Errorf("failed to connect to database: %w", err)
			return
		}
		defer baseDB.Close()
		ctx := context.Background()
		// Obtain a session-level advisory lock. The lock is released when the
		// function scope ends. This lock synchronizes the creation of the
		// templateDB across multiple test binaries, which each have their own
		// instance of `tonce`. This happens when you're running tests from
		// multiple packages in parallel.
		terror = sessionlock.With(ctx, baseDB, "migrate", func() error {
			// If the templateDB already exists, and is marked as a template, it means we
			// migrated successfully and we can exit right now.
			var templateDBExists bool
			query := fmt.Sprintf(
				"SELECT EXISTS (SELECT FROM pg_database WHERE datname = '%s' AND datistemplate = 't')",
				templateConf.Database,
			)
			if err := baseDB.QueryRowContext(ctx, query).Scan(&templateDBExists); err != nil {
				return fmt.Errorf("failed to check if templatedb already exists: %w", err)
			}
			var templateDB *sql.DB
			if templateDBExists {
				return nil
			}

			// If it exists and isn't marked as a template, we failed to migrate it properly, so we
			// should remove it entirely and then recreate it.
			query = fmt.Sprintf("DROP DATABASE IF EXISTS %s", templateConf.Database)
			if _, err := baseDB.ExecContext(ctx, query); err != nil {
				return fmt.Errorf("failed to drop broken template database %s: %w", templateConf.Database, err)
			}

			// Get-or-create a role/user dedicated to connecting to these test databases.
			var roleExists bool
			query = fmt.Sprintf(
				"SELECT EXISTS (SELECT from pg_catalog.pg_roles WHERE rolname = '%s')",
				templateConf.User,
			)
			if err := baseDB.QueryRowContext(ctx, query).Scan(&roleExists); err != nil {
				return fmt.Errorf("failed to detect if role %s exists: %w", templateConf.User, err)
			}
			if !roleExists {
				query = fmt.Sprintf("CREATE ROLE %s", templateConf.User)
				if _, err := baseDB.ExecContext(ctx, query); err != nil {
					return fmt.Errorf("failed to create role %s: %w", templateConf.User, err)
				}
				query = fmt.Sprintf(
					"ALTER ROLE %s WITH LOGIN PASSWORD '%s' NOSUPERUSER NOCREATEDB NOCREATEROLE",
					templateConf.User,
					templateConf.Password,
				)
				if _, err := baseDB.ExecContext(ctx, query); err != nil {
					return fmt.Errorf("failed to alter role and set password for %s: %w", templateConf.User, err)
				}
			}

			// Create the template, owned by the correct user.
			query = fmt.Sprintf("CREATE DATABASE %s OWNER %s", templateConf.Database, templateConf.User)
			if _, err := baseDB.ExecContext(ctx, query); err != nil {
				return fmt.Errorf("failed to create testdb %s: %w", templateConf.Database, err)
			}

			// Connect to the template.
			templateDB, err = templateConf.connect()
			if err != nil {
				return fmt.Errorf("failed to connect to templatedb %s: %w", templateConf.Database, err)
			}
			defer templateDB.Close()

			// Grant all privileges on the template to the test user.
			for _, query = range []string{
				fmt.Sprintf("GRANT ALL PRIVILEGES ON DATABASE %s TO %s", templateConf.Database, templateConf.User),
				fmt.Sprintf("ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL PRIVILEGES ON TABLES TO %s", templateConf.User),
				fmt.Sprintf("ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL PRIVILEGES ON SEQUENCES TO %s", templateConf.User),
			} {
				if _, err := templateDB.ExecContext(ctx, query); err != nil {
					return fmt.Errorf(
						"failed to grant privileges on testdb %s to role %s: %w",
						templateConf.Database,
						templateConf.User,
						err,
					)
				}
			}

			// Apply the Migrator logic one time, when creating the template
			// database.  If this fails, the template database will remain and
			// the developer can connect to it and investigate the failure.
			// Subsequent attempts to create the template will remove it, since
			// it didn't get marked as complete (datistemplate=true).
			if err := migrator.Prepare(ctx, templateDB, templateConf.Config); err != nil {
				return fmt.Errorf("failed to prepare %s: %w", templateConf.Database, err)
			}
			if err := migrator.Migrate(ctx, templateDB, templateConf.Config); err != nil {
				return fmt.Errorf("failed to migrate %s: %w", templateConf.Database, err)
			}

			// Finalize the creation of the template by marking it as a
			// template.
			query = fmt.Sprintf("UPDATE pg_database SET datistemplate=true WHERE datname='%s'", templateConf.Database)
			if _, err := baseDB.ExecContext(ctx, query); err != nil {
				return fmt.Errorf("failed to mark testdb %s as template: %w", templateConf.Database, err)
			}
			return nil
		})

		// If there was any error, make sure the config is unusable. Otherwise,
		// tconfig should point to a valid templateDB!
		if terror != nil {
			tconfig = nil
		} else {
			tconfig = &templateConf
		}
	})

	// If there were any errors creating the template database, log them and
	// immediately fail the test.
	if terror != nil {
		t.Errorf("failed to provision template: %s", terror)
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
		"testdb_tpl_%s_inst_%s",
		templateConf.Hash,
		randomID(),
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

func randomID() string {
	bytes := make([]byte, 4) // 32 random bits means collisons unlikely with less than 4bil
	hash := md5.New()
	_, err := rand.Read(bytes)
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(hash.Sum(bytes))
}
