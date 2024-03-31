package pgtestdb

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"

	"github.com/peterldowns/pgtestdb/internal/once"
	"github.com/peterldowns/pgtestdb/internal/sessionlock"
	"github.com/peterldowns/pgtestdb/migrators/common"
)

const (
	// DefaultRoleUsername is the default name for the role that is created and
	// used to create and connect to each test database.
	DefaultRoleUsername = "pgtdbuser"
	// DefaultRolePassword is the default password for the role that is created and
	// used to create and connect to each test database.
	DefaultRolePassword = "pgtdbpass"
	// DefaultRoleCapabilities is the default set of capabilities for the role
	// that is created and used to create and conect to each test database.
	// This is locked down by default, and will not allow the creation of
	// extensions.
	DefaultRoleCapabilities = "NOSUPERUSER NOCREATEDB NOCREATEROLE"
	// Deprecated: prefer [DefaultRoleUsername].
	TestUser = DefaultRoleUsername
	// Deprecated: prefer [DefaultRolePassword].
	TestPassword = DefaultRolePassword
)

// DefaultRole returns the default Role used to create and connect to the
// template database and each test database.  It is a function, not a struct, to
// prevent accidental overriding.
func DefaultRole() Role {
	return Role{
		Username:     DefaultRoleUsername,
		Password:     DefaultRolePassword,
		Capabilities: DefaultRoleCapabilities,
	}
}

// Config contains the details needed to connect to a postgres server/database.
type Config struct {
	DriverName string // the name of a driver to use when calling sql.Open() to connect to a database, "pgx" (pgx) or "postgres" (lib/pq)
	Host       string // the host of the database, "localhost"
	Port       string // the port of the database, "5433"
	User       string // the user to connect as, "postgres"
	Password   string // the password to connect with, "password"
	Database   string // the database to connect to, "postgres"
	Options    string // URL-formatted additional options to pass in the connection string, "sslmode=disable&something=value"
	// TestRole is the role used to create and connect to the template database
	// and each test database. If not provided, defaults to [DefaultRole].  The
	// capabilities of this role should match the capabilities of the role that
	// your application uses to connect to its database and run migrations.
	TestRole *Role
}

// Role contains the details of a postgres role (user) that will be used
// when creating and connecting to the template and test databases.
type Role struct {
	// The username for the role, defaults to [DefaultRoleUsername].
	Username string
	// The password for the role, defaults to [DefaultRolePassword].
	Password string
	// The capabilities that will be granted to the role, defaults to
	// [DefaultRoleCapabilities].
	Capabilities string
}

// URL returns a postgres connection string in the format
// "postgres://user:password@host:port/database?options=..."
func (c Config) URL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?%s",
		c.User, c.Password, c.Host, c.Port, c.Database, c.Options,
	)
}

// Connect calls `sql.Open()“ and connects to the database.
func (c Config) Connect() (*sql.DB, error) {
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
	// pgtestdb will use the returned Hash to identify a template database. If a
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

// New connects to a postgres server and creates and connects to a fresh
// database instance. This database is prepared and migrated by the given
// migrator, by get-or-creating a template database and then cloning it. This is
// a concurrency-safe primitive. If there is an error creating the database, the
// test will be immediately failed with `t.Fatalf()`.
//
// If this method succeeds, it will `t.Log()` the connection string to the
// created database, so that if your test fails, you can connect to the database
// manually and see what happened.
//
// If this method succeeds and your test succeeds, the database will be removed
// as part of the test cleanup process.
//
// `TB` is a subset of the `testing.TB` testing interface implemented by
// `*testing.T`, `*testing.B`, and `*testing.F`, so you can use pgtestdb to get
// a database for tests, benchmarks, and fuzzes.
func New(t TB, conf Config, migrator Migrator) *sql.DB {
	t.Helper()
	_, db := create(t, conf, migrator)
	return db
}

// TB is a subset of the `testing.TB` testing interface implemented by
// `*testing.T`, `*testing.B`, and `*testing.F`, so you can use pgtestdb to get
// a database for tests, benchmarks, and fuzzes. It contains only the methods
// actually needed by pgtestdb, defined so that we can more easily mock it.
type TB interface {
	Cleanup(func())
	Failed() bool
	Fatalf(format string, args ...any)
	Helper()
	Logf(format string, args ...any)
}

// Custom is like [New] but after creating the new database instance, it closes
// any connections and returns the configuration details of that database so
// that you can connect to it explicitly, potentially via a different SQL
// interface.
func Custom(t TB, conf Config, migrator Migrator) *Config {
	t.Helper()
	config, db := create(t, conf, migrator)
	// Close `*sql.DB` connection that was opened during the creation process so
	// that it the caller can connect to the database in any method of their
	// choosing without interference from this existing connection.
	if err := db.Close(); err != nil {
		t.Fatalf("could not close test database: '%s': %s", config.Database, err)
		return nil // uncreachable
	}
	return config
}

// Helper
// Fatalf
// Fatal
// Logf
// Cleanup
// Failed

func create(t TB, conf Config, migrator Migrator) (*Config, *sql.DB) {
	t.Helper()
	ctx := context.Background()
	baseDB, err := conf.Connect()
	if err != nil {
		t.Fatalf("could not connect to database: %s", err)
		return nil, nil // unreachable
	}

	// From this point onward, all functions assume that `conf.TestRole` is not nil.
	// We default to the
	if conf.TestRole == nil {
		role := DefaultRole()
		conf.TestRole = &role
	}
	if err := ensureUser(ctx, baseDB, conf); err != nil {
		t.Fatalf("could not create pgtestdb user: %s", err)
		return nil, nil // unreachable
	}

	template, err := getOrCreateTemplate(ctx, baseDB, conf, migrator)
	if err != nil {
		t.Fatalf("%s", err)
		return nil, nil // unreachable
	}

	instance, err := createInstance(ctx, baseDB, *template)
	if err != nil {
		t.Fatalf("failed to create instance: %s", err)
		return nil, nil // unreachable
	}
	t.Logf("testdbconf: %s", instance.URL())

	db, err := instance.Connect()
	if err != nil {
		t.Fatalf("failed to connect to instance: %s", err)
		return nil, nil // unreachable
	}

	t.Cleanup(func() {
		// Close the testDB
		if err := db.Close(); err != nil {
			t.Fatalf("could not close test database: '%s': %s", instance.Database, err)
			return // uncreachable
		}
		// If the test failed, leave the instance around for further investigation
		if t.Failed() {
			return
		}
		// Otherwise, remove the instance from the server
		query := fmt.Sprintf(`DROP DATABASE IF EXISTS "%s"`, instance.Database)
		if _, err := baseDB.ExecContext(ctx, query); err != nil {
			t.Fatalf("could not drop test database '%s': %s", instance.Database, err)
			return // unreachable
		}
	})

	// Even if the template previously existed, verify that it was created
	// successfully.
	// This way if there was ever a mistake or problem creating the template, a
	// test will find out at the site of `pgtestdb.New` rather than later in the
	// test due to unexpected content in the database.
	//
	// Assumption: verification is >>> faster than performing the migrations,
	// and is therefore safe to run at the beginning of each test.
	if err := migrator.Verify(ctx, db, *instance); err != nil {
		t.Fatalf("test database failed verification %s: %w", instance.Database, err)
		return nil, nil // unreachable
	}

	return instance, db
}

// user is used to guarantee that each testdb user/role is only get-or-created
// at most once per program. Different calls to pgtestdb can specify different
// roles, but each will be get-or-created at most one time per program, and will
// be created only once no matter how many different programs or test suites run
// at once, thanks to the use of session locks.
var users once.Map[string, any] = once.NewMap[string, any]() //nolint:gochecknoglobals

func ensureUser(
	ctx context.Context,
	baseDB *sql.DB,
	conf Config,
) error {
	username := conf.TestRole.Username
	_, err := users.Set(username, func() (*any, error) {
		return nil, sessionlock.With(ctx, baseDB, username, func(conn *sql.Conn) error {
			// Get-or-create a role/user dedicated to connecting to these test databases.
			var roleExists bool
			query := "SELECT EXISTS (SELECT from pg_catalog.pg_roles WHERE rolname = $1)"
			if err := conn.QueryRowContext(ctx, query, username).Scan(&roleExists); err != nil {
				return fmt.Errorf("failed to detect if role %s exists: %w", username, err)
			}
			if roleExists {
				return nil
			}
			if !roleExists {
				query = fmt.Sprintf(`CREATE ROLE "%s"`, username)
				if _, err := conn.ExecContext(ctx, query); err != nil {
					return fmt.Errorf("failed to create role %s: %w", username, err)
				}
				query = fmt.Sprintf(
					`ALTER ROLE "%s" WITH LOGIN PASSWORD '%s' %s`,
					username,
					conf.TestRole.Password,
					conf.TestRole.Capabilities,
				)
				if _, err := conn.ExecContext(ctx, query); err != nil {
					return fmt.Errorf("failed to set password and capabilities for '%s': %w", username, err)
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

var templates once.Map[string, templateState] = once.NewMap[string, templateState]() //nolint:gochecknoglobals

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
	mhash, err := migrator.Hash()
	if err != nil {
		return nil, fmt.Errorf("failed to calculate template hash: %w", err)
	}
	// The migrator Hash() implementation is included, along with the role
	// details, so that if the user runs tests in parallel with different role
	// information, they each get their own database.
	hash := common.NewRecursiveHash(
		common.Field("Username", dbconf.TestRole.Username),
		common.Field("Password", dbconf.TestRole.Password),
		common.Field("Capabilities", dbconf.TestRole.Capabilities),
		common.Field("MigratorHash", mhash),
	).String()

	return templates.Set(hash, func() (*templateState, error) {
		// This function runs once per program, but only synchronizes access
		// within a single program. When running larger test suites, each
		// package's tests may run in parallel, which means this does not
		// perfectly synchronize interaction with the database.
		state := templateState{}
		state.hash = hash
		state.conf = dbconf
		state.conf.TestRole = dbconf.TestRole
		state.conf.User = dbconf.TestRole.Username
		state.conf.Password = dbconf.TestRole.Password
		state.conf.Database = fmt.Sprintf("testdb_tpl_%s", hash)
		// sessionlock synchronizes the creation of the template with a
		// session-scoped advisory lock.
		err := sessionlock.With(ctx, baseDB, state.conf.Database, func(conn *sql.Conn) error {
			return ensureTemplate(ctx, conn, migrator, state)
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
	state templateState,
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

	// Connect to the template.
	template, err := state.conf.Connect()
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
// https://github.com/peterldowns/pgtestdb#testdbmigrator
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
