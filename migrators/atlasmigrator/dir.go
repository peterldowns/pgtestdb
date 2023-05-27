package atlasmigrator

import (
	"context"
	"database/sql"

	"github.com/peterldowns/testdb"
	"github.com/peterldowns/testdb/migrators/common"
)

// NewDirMigrator returns a [DirMigrator], which is a testdb.Migrator that
// uses the `atlas` CLI tool to perform migrations.
//
//	atlas migrate apply --url $DB --dir file://$migrationsDirPath
func NewDirMigrator(
	migrationsDirPath string,
) *DirMigrator {
	return &DirMigrator{
		MigrationsDirPath: migrationsDirPath,
	}
}

// DirMigrator is a testdb.Migrator that uses the `atlas` CLI
// tool to perform migrations.
//
//	atlas migrate apply --url $DB --dir file://$migrationsDirPath
//
// DirMigrator requires that it runs in an environment where the `atlas` CLI is
// in the $PATH. It shells out to that program to perform its migrations,
// as recommended by the Atlas maintainers.
//
// DirMigrator does not perform any Verify() or Prepare() logic.
type DirMigrator struct {
	MigrationsDirPath string
}

func (m *DirMigrator) Hash() (string, error) {
	return common.HashDir(m.MigrationsDirPath)
}

// Migrate shells out to the `atlas` CLI program to migrate the template
// database.
//
//	atlas migrate apply --url $DB --dir file://$migrationsDirPath
func (m *DirMigrator) Migrate(
	ctx context.Context,
	_ *sql.DB,
	templateConf testdb.Config,
) error {
	_, err := common.Execute(ctx, nil,
		"atlas",
		"migrate",
		"apply",
		"--url",
		templateConf.URL(),
		"--dir",
		"file://"+m.MigrationsDirPath,
	)
	return err
}

// Prepare is a no-op method.
func (m *DirMigrator) Prepare(
	_ context.Context,
	_ *sql.DB,
	_ testdb.Config,
) error {
	return nil
}

// Verify is a no-op method.
func (m *DirMigrator) Verify(
	_ context.Context,
	_ *sql.DB,
	_ testdb.Config,
) error {
	return nil
}
