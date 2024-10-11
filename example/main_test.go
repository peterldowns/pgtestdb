package main_test

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/peterldowns/testy/assert"

	"github.com/peterldowns/pgtestdb"
	"github.com/peterldowns/pgtestdb/migrators/common"
	"github.com/peterldowns/pgtestdb/migrators/goosemigrator"
)

//go:embed migrations/*.sql
var exampleFS embed.FS

type MyMigrator struct {
	goosemigrator.GooseMigrator
}

func (m *MyMigrator) Prepare(ctx context.Context, db *sql.DB, conf pgtestdb.Config) error {
	// If your migrations assume that you have manually enabled some extensions
	// before you attempt to run them, you can match that behavior by
	// "overriding" the Prepare() method from an existing migrator. Here, we
	// enable the "hstore" extension.  pgtestdb calls `Prepare()` right before
	// it calls `Migrate()`, so your migrations can just assume that "hstore" is
	// enabled (see the example migrations.)
	_, err := db.ExecContext(ctx, "CREATE EXTENSION hstore;")
	if err != nil {
		return fmt.Errorf("db failed to activate hstore: %w", err)
	}
	// Some extensions like "postgis" require the current role to be `SUPERUSER` in order
	// to enable them.
	//
	// pgtestdb will call `Prepare()` and `Migrate()` with the exact same connection information,
	//	- username = `conf.TestRole.Username`, defaults to `pgtestdb.DefaultRoleUsername`
	//  - password = `conf.TestRole.Password`, defaults to `pgtestdb.DefaultRolePassword`
	//  - capabilities = `conf.TestRole.Capabilities`, defaults to `pgtestdb.DefaultRoleCapabilities`
	//
	// Importantly, the default capabilities are `"NOSUPERUSER NOCREATEDB NOCREATEROLE"` — which
	// is probably the same set of capabilities your app uses to connect to your production database.
	// But `NOSUPERUSER` will prevent us from activiating "postgis", because:
	//	 When creating a test database, pgtestdb will:
	//	 1. create a new role `conf.TestRole.Username` with capabilities `conf.TestRole.Capabilities`
	//	   - username defaults to "pgtdbuser"
	//	   - capabilities defaults to "NOSUPERUSER NOCREATEDB NOCREATEROLE"
	//	 2. if it doesn't already exist, create the template database
	//	   - the template database is owned by `conf.TestRole.Username`
	//	 	 - 2.a: connect to the template database as `conf.TestRole.Username`
	//		 - 2.b: call Prepare() with this same connection
	//	   - 2.c: call Migrate() with this same connection
	//	 3. create an instance database based on this template
	//	 	- the instance database is owned by `conf.TestRole.Username`
	//
	// Enabling "postgis" requires the current role/user to have ADMINISTRATOR priviliges.
	// Just trying to activate it will fail...
	_, err = db.ExecContext(ctx, "CREATE EXTENSION postgis;")
	if err != nil {
		// ... so instead, you have two options. You can connect as a superuser,
		// and activate the extension using the superuser connection:
		superuserConnToTemplateConf := conf
		superuserConnToTemplateConf.User = "postgres"
		superuserConnToTemplateConf.Password = "password"
		suDB, err := superuserConnToTemplateConf.Connect()
		if err != nil {
			return fmt.Errorf("suDB failed to connect: %w", err)
		}
		defer suDB.Close()
		_, err = suDB.ExecContext(ctx, "CREATE EXTENSION postgis;")
		if err != nil {
			return fmt.Errorf("suDB failed to activate postgis: %w", err)
		}
		// Or, you can avoid this entirely by setting `conf.TestRole.Capabilities` to
		// something like `NOCREATEDB NOCREATEROLE`, so that it still has
		// the `SUPERUSER` capability. This means that each test gets a database connection
		// with the `SUPERUSER` capability, which is potentially different than how your app
		// connects to its database in production (usually doesn't have `SUPERUSER`), but
		// if you're willing to accept that difference, it's an easier change.
	}
	return m.GooseMigrator.Prepare(ctx, db, conf)
}

func (m *MyMigrator) Hash() (string, error) {
	wrapped, err := m.GooseMigrator.Hash()
	if err != nil {
		return "", err
	}
	hash := common.NewRecursiveHash(
		common.Field("Migrations", wrapped),
		common.Field("Prepare", `
			you have to change this string if you modify the sql statements
			executed in Prepare() or otherwise pgtestdb won't know that it
			should create a new template.
		`),
	).String()
	return hash, nil
}

func TestCustomPrepareMethod(t *testing.T) {
	t.Parallel()
	conf := pgtestdb.Config{
		DriverName: "pgx",
		User:       "postgres",
		Password:   "password",
		Host:       "localhost",
		Port:       "5433",
		Options:    "sslmode=disable",
	}
	migrator := &MyMigrator{
		*goosemigrator.New("migrations", goosemigrator.WithFS(exampleFS)),
	}
	db := pgtestdb.New(t, conf, migrator)

	var name string
	err := db.QueryRow("SELECT h['name'] FROM myhstoredata;").Scan(&name)
	assert.Nil(t, err)
	assert.Equal(t, "Peter", name)
}
