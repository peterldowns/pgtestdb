package sessionlock

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // pgx driver for postgres
	"github.com/peterldowns/testy/assert"
	"github.com/peterldowns/testy/check"

	"github.com/peterldowns/testdb/internal/withdb"
)

func TestWithSessionLock(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	check.Nil(t, withdb.WithDB(ctx, "pgx", func(db *sql.DB) error {
		var counter int32
		var wg sync.WaitGroup

		for i := 0; i < 10; i++ {
			wg.Add(1)

			go func() {
				defer wg.Done()
				err := With(ctx, db, "test-with-session-lock", func(_ *sql.Conn) error {
					newCounter := atomic.AddInt32(&counter, 1)
					check.Equal(t, int32(1), newCounter)

					time.Sleep(time.Millisecond * 10)

					newCounter = atomic.AddInt32(&counter, -1)
					check.Equal(t, int32(0), newCounter)

					return nil
				})

				check.Nil(t, err)
			}()
		}
		wg.Wait()
		return nil
	}))
}

func TestWithReturnsErrorsFromCallback(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	check.Nil(t, withdb.WithDB(ctx, "pgx", func(db *sql.DB) error {
		// This error should be the same error returned by conn
		err := With(ctx, db, "example", func(conn *sql.Conn) error {
			_, err := conn.ExecContext(ctx, "select broken query")
			return err
		})
		check.NotEqual(t, nil, err)
		return nil
	}))
}

func TestWithReturnsUnlockErrors(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	check.Nil(t, withdb.WithDB(ctx, "pgx", func(db *sql.DB) error {
		err := With(ctx, db, "example", func(conn *sql.Conn) error {
			err := conn.Close()
			if !check.Nil(t, err) {
				return fmt.Errorf("inner: %w", err)
			}
			return nil
		})
		assert.NotEqual(t, nil, err)
		msgs := strings.Split(err.Error(), "\n")
		check.Equal(t, []string{
			"sessionlock(example) failed to unlock: sql: connection is already closed",
			"sessionlock(example) failed to close conn: sql: connection is already closed",
		}, msgs)
		return nil
	}))
}
