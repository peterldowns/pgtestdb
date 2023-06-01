package testdb

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/peterldowns/testdb/internal/once"
	"github.com/peterldowns/testdb/internal/sessionlock"
)

const (
	// TestUser is the username for connecting to each test database.
	TestUser = "testdbuser"
	// TestPassword is the password for connecting to each test database.
	TestPassword = "password"
)

// Config contains the details needed to connect to a postgres server/database.
type Config struct {
	DriverName string
	Host       string
	Port       string
	User       string
	Password   string
	Database   string
	Options    string
}

// URL returns a postgres:// connection string based on the details of this
// config.
func (c Config) URL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?%s",
		c.User, c.Password, c.Host, c.Port, c.Database, c.Options,
	)
}

func (c Config) connect() (*sql.DB, error) {
	db, err := sql.Open(c.DriverName, c.URL())
	if err != nil {
		return nil, err
	}
	return db, nil
}

// A Migrator is necessary to provision and verify the database that will be used as as template
// for each test.
type Migrator interface {
	// Hash should return a unique identifier derived from the state of the database
	// after it has been fully migrated. For instance, it may return a hash of all
	// of the migration names and contents.
	//
	// testdb will use the returned Hash to identify a template database. If a
	// Migrator returns a Hash that has already been used to create a template
	// database, it is assumed that the database need not be recreated since it
	// would result in the same schema and data.
	Hash() (string, error)

	// Prepare should perform any plugin or extension installations necessary to
	// make the database ready for the migrations. For instance, you may want to
	// enable certain extensions like `trigram` or `pgcrypto`, or creating or
	// altering certain roles and permissions.
	// Prepare will be given a *sql.DB connected to the template database.
	Prepare(context.Context, *sql.DB, Config) error

	// Migrate is a function that actually performs the schema and data
	// migrations to provision a template database. The connection given to this
	// function is to an entirely new, empty, database. Migrate will be called
	// only once, when the template database is being created.
	Migrate(context.Context, *sql.DB, Config) error

	// Verify is called each time you ask for a new test database instance. It
	// should be cheaper than the call to Migrate(), and should return nil iff
	// the database is in the correct state. An example implementation would be
	// to check that all the migrations have been marked as applied, and
	// otherwise return an error.
	Verify(context.Context, *sql.DB, Config) error
}

// New returns a fresh database, prepared and migrated by the given migrator, by
// get-or-creating a template database and then cloning it. This is a
// concurrency-safe primitive. If there is an error creating the database, the
// test will be immediately failed with `t.Fatal()`.
//
// If this method succeeds, it will `t.Log()` the connection string to the
// created database, so that if your test fails, you can connect to the database
// manually and see what happened.
//
// If this method succeeds and your test succeeds, the database will be removed
// as part of the test cleanup process.
func New(t *testing.T, conf Config, migrator Migrator) *sql.DB {
	t.Helper()
	ctx := context.Background()
	baseDB, err := conf.connect() // TODO: allow dialect to support non-pgx
	if err != nil {
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
		query := fmt.Sprintf(`DROP DATABASE IF EXISTS "%s"`, instance.Database)
		if _, err := baseDB.ExecContext(ctx, query); err != nil {
			t.Fatalf("could not drop test database '%s': %s", instance.Database, err)
		}
	})

	// Even if the template previously existed, verify that it was created
	// successfully.
	// This way if there was ever a mistake or problem creating the template, a
	// test will find out at the site of `testdb.New` rather than later in the
	// test due to unexpected content in the database.
	//
	// Assumption: verification is >>> faster than performing the migrations,
	// and is therefore safe to run at the beginning of each test.
	if err := migrator.Verify(ctx, db, *instance); err != nil {
		t.Fatal(fmt.Errorf("test database failed verification %s: %w", instance.Database, err))
	}

	return db
}

// user is used to guarantee that the testdb user/role is only get-or-created at
// most once per program.
var user once.Var[any] = once.NewVar[any]() //nolint:gochecknoglobals

func ensureUser(
	ctx context.Context,
	baseDB *sql.DB,
) error {
	_, err := user.Set(func() (*any, error) {
		return nil, sessionlock.With(ctx, baseDB, "testdb-user", func(conn *sql.Conn) error {
			// Get-or-create a role/user dedicated to connecting to these test databases.
			var roleExists bool
			query := "SELECT EXISTS (SELECT from pg_catalog.pg_roles WHERE rolname = $1)"
			if err := conn.QueryRowContext(ctx, query, TestUser).Scan(&roleExists); err != nil {
				return fmt.Errorf("failed to detect if role %s exists: %w", TestUser, err)
			}
			if roleExists {
				return nil
			}
			if !roleExists {
				query = fmt.Sprintf(`CREATE ROLE "%s"`, TestUser)
				if _, err := conn.ExecContext(ctx, query); err != nil {
					return fmt.Errorf("failed to create role %s: %w", TestUser, err)
				}
				query = fmt.Sprintf(
					`ALTER ROLE "%s" WITH LOGIN PASSWORD '%s' NOSUPERUSER NOCREATEDB NOCREATEROLE`,
					TestUser,
					TestPassword,
				)
				if _, err := conn.ExecContext(ctx, query); err != nil {
					return fmt.Errorf("failed to alter role and set password for %s: %w", TestUser, err)
				}
			}
			return nil
		})
	})
	return err
}

// templateState keeps the state of a single template, so that each program only
// attempts to create/migrate the template at most once.
type templateState struct {
	conf Config
	hash string
}

var states once.Map[string, templateState] = once.NewMap[string, templateState]() //nolint:gochecknoglobals

// getOrCreateTemplate will get-or-create a template, synchronizing at
// the golang level (with the states map, so that each template is
// get-or-created at most once) and at the postgres level (with advisory locks,
// so that there are no conflicts between multiple programs trying to create the
// template.)
//
// If there was a database error during template creation, the program that
// attempted the creation will set state.error, so subsequent attempts to access
// the template from within the same golang program will just return that error.
//
// This means that:
// - migrations are only run once per template per golang program / package under test.
// - you don't need to manually clear out "broken" templates between test suite runs.
func getOrCreateTemplate(
	ctx context.Context,
	baseDB *sql.DB,
	dbconf Config,
	migrator Migrator,
) (*templateState, error) {
	hash, err := migrator.Hash()
	if err != nil {
		return nil, fmt.Errorf("failed to calculate template hash: %w", err)
	}
	return states.Set(hash, func() (*templateState, error) {
		// This function runs once per program, but only synchronizes access
		// within a single program. When running larger test suites, each
		// package's tests may run in parallel, which means this does not
		// perfectly synchronize interaction with the database.
		state := templateState{}
		state.hash = hash
		state.conf = dbconf
		state.conf.User = TestUser
		state.conf.Password = TestPassword
		state.conf.Database = fmt.Sprintf("testdb_tpl_%s", hash)
		// sessionlock synchronizes the creation of the template with a
		// session-scoped advisory lock.
		err := sessionlock.With(ctx, baseDB, state.conf.Database, func(conn *sql.Conn) error {
			return ensureTemplate(ctx, conn, migrator, &state)
		})
		if err != nil {
			return nil, err
		}
		return &state, nil
	})
}

// ensureTemplate uses the 'datistemplate' column to mark a template as having
// been successfully created, and does not set 'datistemplate = true' until the
// database has been successfully created and migrated. If it finds a template
// database where 'datistemplate = false', it drops and then attempts to
// recreate that database.
func ensureTemplate(
	ctx context.Context,
	conn *sql.Conn,
	migrator Migrator,
	state *templateState,
) error {
	// If the template database already exists, and is marked as a template,
	// there is no more work to be done.
	var templateExists bool
	query := "SELECT EXISTS (SELECT FROM pg_database WHERE datname = $1 AND datistemplate = true)"
	if err := conn.QueryRowContext(ctx, query, state.conf.Database).Scan(&templateExists); err != nil {
		return fmt.Errorf("failed to check if template %s already exists: %w", state.conf.Database, err)
	}
	if templateExists {
		return nil
	}

	// If the template database already exists, but it is not marked as a
	// template, there was a failure at some point during the creation process
	// so it needs to be deleted.
	query = fmt.Sprintf(`DROP DATABASE IF EXISTS "%s"`, state.conf.Database)
	if _, err := conn.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("failed to drop broken template %s: %w", state.conf.Database, err)
	}

	query = fmt.Sprintf(`CREATE DATABASE "%s" OWNER "%s"`, state.conf.Database, state.conf.User)
	if _, err := conn.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("failed to create template %s: %w", state.conf.Database, err)
	}

	// Grant all privileges on the template to the test user.
	for _, query = range []string{
		fmt.Sprintf(`GRANT ALL PRIVILEGES ON DATABASE "%s" TO "%s"`, state.conf.Database, state.conf.User),
		fmt.Sprintf(`ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL PRIVILEGES ON TABLES TO "%s"`, state.conf.User),
		fmt.Sprintf(`ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL PRIVILEGES ON SEQUENCES TO "%s"`, state.conf.User),
	} {
		if _, err := conn.ExecContext(ctx, query); err != nil {
			return fmt.Errorf(
				"failed to grant privileges on template %s to role %s: %w",
				state.conf.Database,
				state.conf.User,
				err,
			)
		}
	}

	// Connect to the template.
	template, err := state.conf.connect()
	if err != nil {
		return fmt.Errorf("failed to connect to template %s: %w", state.conf.Database, err)
	}
	defer template.Close()

	// Apply the Migrator logic one time, when creating the template. If this
	// fails, the template will remain and the developer can connect to it and
	// investigate the failure. Subsequent attempts to create the template will
	// remove it, since it didn't get marked as complete (datistemplate=true).
	if err := migrator.Prepare(ctx, template, state.conf); err != nil {
		return fmt.Errorf("failed to migrator.Prepare template %s: %w", state.conf.Database, err)
	}
	if err := migrator.Migrate(ctx, template, state.conf); err != nil {
		return fmt.Errorf("failed to migrator.Migrate template %s: %w", state.conf.Database, err)
	}

	// Finalize the creation of the template by marking it as a
	// template.
	query = "UPDATE pg_database SET datistemplate = true WHERE datname=$1"
	if _, err := conn.ExecContext(ctx, query, state.conf.Database); err != nil {
		return fmt.Errorf("failed to confirm template %s: %w", state.conf.Database, err)
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
		return nil, fmt.Errorf("failed to create instance from template %s: %w", template.conf.Database, err)
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

// NoopMigrator fulfills the Migrator interface but does absolutely nothing.
// You can use this to get empty databases in your tests, or if you are trying
// out testdb and aren't sure which migrator to use yet.
//
// For more documentation on migrators, see
// https://github.com/peterldowns/testdb#testdbmigrator
type NoopMigrator struct{}

func (NoopMigrator) Hash() (string, error) {
	return "noop", nil
}

func (NoopMigrator) Prepare(_ context.Context, _ *sql.DB, _ Config) error {
	return nil
}

func (NoopMigrator) Migrate(_ context.Context, _ *sql.DB, _ Config) error {
	return nil
}

func (NoopMigrator) Verify(_ context.Context, _ *sql.DB, _ Config) error {
	return nil
}
