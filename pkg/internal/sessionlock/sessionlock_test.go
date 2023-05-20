package sessionlock

import (
	"context"
	"database/sql"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/lib/pq"

	"github.com/peterldowns/testdb/pkg/internal/withdb"

	"github.com/stretchr/testify/require"
)

func TestWithSessionLock(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	require.NoError(t, withdb.WithDB(ctx, func(db *sql.DB) {
		var counter int32
		var wg sync.WaitGroup

		for i := 0; i < 10; i++ {
			wg.Add(1)

			go func() {
				defer wg.Done()
				err := With(ctx, db, "test-with-session-lock", func() error {
					newCounter := atomic.AddInt32(&counter, 1)
					require.Equal(t, int32(1), newCounter)

					time.Sleep(time.Millisecond * 10)

					newCounter = atomic.AddInt32(&counter, -1)
					require.Equal(t, int32(0), newCounter)

					return nil
				})

				require.NoError(t, err)
			}()
		}
		wg.Wait()
	}))
}
