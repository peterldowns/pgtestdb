package once_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/peterldowns/testy/assert"
	"github.com/peterldowns/testy/check"

	"github.com/peterldowns/pgtestdb/internal/once"
)

func TestVar(t *testing.T) {
	t.Parallel()
	x := newMutexCounter()
	onceInt := once.NewVar[int]()
	assert.NoFailures(t, func() {
		val, err := onceInt.Get()
		check.Equal(t, nil, val)
		check.Equal(t, nil, err)
		check.Equal(t, 0, x.Read())
	})

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			val, err := onceInt.Set(func() (*int, error) {
				x.Add(1)
				return nil, fmt.Errorf("problem initializing")
			})
			check.Equal(t, nil, val)
			check.Error(t, err)
			check.Equal(t, 1, x.Read())
		}()
	}
	wg.Wait()
	check.Equal(t, 1, x.Read())
}

func TestMap(t *testing.T) {
	t.Parallel()
	x := newMutexCounter()
	onceMap := once.NewMap[string, string]()
	key := "hello"
	assert.NoFailures(t, func() {
		val, err := onceMap.Get(key)
		check.Equal(t, nil, val)
		check.Equal(t, nil, err)
		check.Equal(t, 0, x.Read())
	})

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			val, err := onceMap.Set(key, func() (*string, error) {
				x.Add(1)
				val := "world"
				return &val, nil
			})
			check.Nil(t, err)
			if check.NotEqual(t, nil, val) {
				check.Equal(t, "world", *val)
			}
			check.Equal(t, 1, x.Read())
		}()
	}
	wg.Wait()
	check.Equal(t, 1, x.Read())
}

// mutexCounter is a concurrency-safe counter needed for testing that the other
// "concurrency-safe" code is actually, well, concurrency-safe.
type mutexCounter struct {
	mu     *sync.RWMutex
	number int
}

func newMutexCounter() *mutexCounter {
	return &mutexCounter{&sync.RWMutex{}, 0}
}

func (c *mutexCounter) Add(num int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.number += num
}

func (c *mutexCounter) Read() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.number
}
