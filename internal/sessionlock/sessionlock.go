// sessionlock package provides support for application level distributed locks via advisory
// locks in PostgreSQL.
//
// - https://www.postgresql.org/docs/current/explicit-locking.html#ADVISORY-LOCKS
// - https://samu.space/distributed-locking-with-postgres-advisory-locks/
package sessionlock

import (
	"context"
	"database/sql"
	"fmt"
	"hash/crc32"

	"github.com/peterldowns/pgtestdb/internal/multierr"
)

// IDPrefix is prepended to any given lock name when computing the integer lock
// ID, to help prevent collisions with other clients that may be acquiring their
// own locks.
const IDPrefix string = "sessionlock-"

// ID consistently hashes a string to unique integer that can be used with
// pg_advisory_lock() and pg_advisory_unlock().
func ID(name string) uint32 {
	return crc32.ChecksumIEEE([]byte(IDPrefix + name))
}

// With will open a connection to the `db`, acquire an advisory lock, use that
// connection to acquire an advisory lock, then call your `cb`, then release the
// advisory lock.
func With(ctx context.Context, db *sql.DB, lockName string, cb func(*sql.Conn) error) (final error) {
	id := ID(lockName)
	lockQuery := fmt.Sprintf("SELECT pg_advisory_lock(%d)", id)
	unlockQuery := fmt.Sprintf("SELECT pg_advisory_unlock(%d)", id)

	// Uses a *sql.Conn here to guarantee that lock() and unlock() happen in the
	// same session.
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("sessionlock(%s) failed to open conn: %w", lockName, err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			final = multierr.Join(final, fmt.Errorf("sessionlock(%s) failed to close conn: %w", lockName, err))
		}
	}()

	if _, err := conn.ExecContext(ctx, lockQuery); err != nil {
		return err
	}
	defer func() {
		if _, err := conn.ExecContext(ctx, unlockQuery); err != nil {
			final = multierr.Join(final, fmt.Errorf("sessionlock(%s) failed to unlock: %w", lockName, err))
		}
	}()
	return cb(conn)
}
