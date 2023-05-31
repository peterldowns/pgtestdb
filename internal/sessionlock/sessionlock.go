package sessionlock

import (
	"context"
	"database/sql"
	"fmt"
	"hash/crc32"
)

// IDPrefix is prepended to any given lock name when computing the integer lock
// ID, to help prevent collisions with other clients that may be acquiring their
// own locks.
const IDPrefix string = "sessionlock-"

func ID(name string) uint32 {
	return crc32.ChecksumIEEE([]byte(IDPrefix + name))
}

// This package provides support for application level distributed locks via advisory
// locks in PostgreSQL.
//
// https://www.postgresql.org/docs/current/explicit-locking.html#ADVISORY-LOCKS
// https://samu.space/distributed-locking-with-postgres-advisory-locks/

// With will open a connection to the `db`, acquire an advisory lock, use that
// connection to acquire an advisory lock, then call your `cb`, then release the
// advisory lock.
func With(ctx context.Context, db *sql.DB, lockName string, cb func() error) error {
	// We use *sql.Conn here because it's bound to single DB session, while
	// *sql.DB is a pool of connections that the driver manages for us. This
	// ensures that we have a persistent session with a lock for the duration of
	// the callback, and only use a single connection.
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err := conn.Close(); err != nil {
			panic(err)
		}
	}()

	unlock, err := New(ctx, conn, lockName)
	if err != nil {
		return err
	}
	defer func() {
		if err := unlock(); err != nil {
			panic(err)
		}
	}()
	// TODO: why not call cb() with the conn?
	return cb()
}

// New creates a new advisory lock given a `conn`, and returns a function that
// will use the same `conn` to release the lock. The lock is automatically
// released if the `conn` is closed.
func New(ctx context.Context, conn *sql.Conn, lockName string) (func() error, error) {
	id := ID(lockName)
	qs := fmt.Sprintf("SELECT pg_advisory_lock(%d)", id)
	if _, err := conn.ExecContext(ctx, qs); err != nil {
		return nil, err
	}
	return func() error {
		qs := fmt.Sprintf("SELECT pg_advisory_unlock(%d)", id)
		// TODO: why scan the result to success, and not just Exec to unlock?
		var success bool
		if err := conn.QueryRowContext(ctx, qs).Scan(&success); err != nil {
			return err
		}
		if !success {
			return fmt.Errorf("lock not held")
		}
		return nil
	}, nil
}
