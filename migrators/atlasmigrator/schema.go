package atlasmigrator

import (
	"context"
	"database/sql"

	"github.com/peterldowns/testdb"
	"github.com/peterldowns/testdb/migrators/common"
)

// NewSchemaMigrator returns a [SchemaMigrator], which is a testdb.Migrator that
// uses the `atlas` CLI tool to perform migrations.
//
//	atlas schema apply --auto-approve --url $DB --to file://$schemaFilePath
func NewSchemaMigrator(
	schemaFilePath string,
) *SchemaMigrator {
	return &SchemaMigrator{SchemaFilePath: schemaFilePath}
}

// SchemaMigrator is a testdb.Migrator that uses the `atlas` CLI tool to perform
// migrations.
//
//	atlas schema apply --auto-approve --url $DB --to file://$schemaFilePath
//
// SchemaMigrator requires that it runs in an environment where the `atlas` CLI is
// in the $PATH. It shells out to that program to perform its migrations,
// as recommended by the Atlas maintainers.
//
// SchemaMigrator does not perform any Verify() or Prepare() logic.
type SchemaMigrator struct {
	SchemaFilePath string
}

// Hash returns the md5 hash of the schema file.
func (m *SchemaMigrator) Hash() (string, error) {
	return common.HashFile(m.SchemaFilePath)
}

// Migrate shells out to the `atlas` CLI program to migrate the template
// database.
//
//	atlas schema apply --auto-approve --url $DB --to file://$schemaFilePath
func (m *SchemaMigrator) Migrate(
	ctx context.Context,
	_ *sql.DB,
	templateConf testdb.Config,
) error {
	_, err := common.Execute(ctx, nil,
		"atlas",
		"schema",
		"apply",
		"--auto-approve",
		"--url",
		templateConf.URL(),
		"--to",
		"file://"+m.SchemaFilePath,
	)
	return err
}

// Prepare is a no-op method.
func (m *SchemaMigrator) Prepare(
	_ context.Context,
	_ *sql.DB,
	_ testdb.Config,
) error {
	return nil
}

// Verify is a no-op method.
func (m *SchemaMigrator) Verify(
	_ context.Context,
	_ *sql.DB,
	_ testdb.Config,
) error {
	return nil
}
