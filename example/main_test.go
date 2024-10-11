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
	"github.com/peterldowns/pgtestdb/migrators/ternmigrator"
)

//go:embed migrations/*.sql
var exampleFS embed.FS

type MyMigrator struct {
	ternmigrator.TernMigrator
}

func (m *MyMigrator) Migrate(ctx context.Context, db *sql.DB, conf pgtestdb.Config) error {
	err := m.TernMigrator.Migrate(ctx, db, conf)
	if err != nil {
		return err
	}
	for i := 0; i < 10; i++ {
		name := fmt.Sprintf("Person %d", i)
		_, err := db.ExecContext(ctx, "INSERT INTO people VALUES ($1);", name)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *MyMigrator) Hash() (string, error) {
	wrapped, err := m.TernMigrator.Hash()
	if err != nil {
		return "", err
	}
	hash := common.NewRecursiveHash(
		common.Field("Migrations", wrapped),
		common.Field("FixtureData", `
			you have to change this string if you modify the sql statements
			executed in Migrate() or otherwise pgtestdb won't know that it
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
		*ternmigrator.New("migrations", ternmigrator.WithFS(exampleFS)),
	}
	db := pgtestdb.New(t, conf, migrator)

	var count int
	err := db.QueryRow("SELECT count(*) from people").Scan(&count)
	assert.Nil(t, err)
	assert.Equal(t, 10, count)
}
