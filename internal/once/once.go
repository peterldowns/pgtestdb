// safeonce contains helpers for constructing type-safe, concurrency-safe values
// that are only ever initialized once, and can potentially return an error.
package once

import "sync"

// Map is a type-safe and concurrency-safe implementation of a map where each
// entry is initialized a single time.
type Map[K comparable, V any] interface {
	// Set initializes the key K at most one time, and returns the result.
	Set(K, func() (*V, error)) (*V, error)
	// Get returns the initialization result associated with the key K.
	// If K has not yet been initialized, the result will be (<nil>, <nil>).
	Get(K) (*V, error)
}

// Var is a type-safe and concurrency-safe wrapper for a value that is
// initialized a single time.
type Var[T any] interface {
	Set(func() (*T, error)) (*T, error)
	Get() (*T, error)
}

// NewMap returns a type-safe and concurrency-safe map of keys that
// can each be initialized a single time.
func NewMap[K comparable, V any]() Map[K, V] {
	return &smap[K, V]{}
}

type entry[V any] struct {
	data *V
	err  error
}

type smap[K comparable, V any] struct {
	onces sync.Map // map[K]*sync.Once
	data  sync.Map // map[K]safemapEntry
}

func (sm *smap[K, V]) Set(key K, f func() (*V, error)) (*V, error) {
	onceRaw, _ := sm.onces.LoadOrStore(key, &sync.Once{})
	once := onceRaw.(*sync.Once)
	once.Do(func() {
		res, err := f()
		sm.data.Store(key, entry[V]{
			data: res,
			err:  err,
		})
	})
	return sm.Get(key)
}

func (sm *smap[K, V]) Get(key K) (*V, error) {
	rawState, _ := sm.data.Load(key)
	state, _ := rawState.(entry[V])
	return state.data, state.err
}

func NewVar[T any]() Var[T] {
	return &svar[T]{}
}

type svar[T any] struct {
	data *T
	once sync.Once
	mu   sync.RWMutex
	err  error
}

func (sv *svar[T]) withReadLock(f func()) {
	sv.mu.RLock()
	defer sv.mu.RUnlock()
	f()
}

func (sv *svar[T]) withWriteLock(f func()) {
	sv.mu.Lock()
	defer sv.mu.Unlock()
	f()
}

func (sv *svar[T]) Set(f func() (*T, error)) (*T, error) {
	var x *T
	var e error
	sv.withWriteLock(func() {
		sv.once.Do(func() {
			sv.data, sv.err = f()
		})
		x = sv.data
		e = sv.err
	})
	return x, e
}

func (sv *svar[T]) Get() (*T, error) {
	var out *T
	var err error
	sv.withReadLock(func() {
		out = sv.data
		err = sv.err
	})
	return out, err
}
