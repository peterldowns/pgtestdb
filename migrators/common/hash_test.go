package common_test

import (
	"testing"

	"github.com/peterldowns/testy/assert"

	"github.com/peterldowns/pgtestdb/migrators/common"
)

func TestRecursiveHash(t *testing.T) {
	t.Parallel()
	hash1 := common.NewRecursiveHash()
	hash2 := common.NewRecursiveHash()
	assert.Equal(t, hash1.String(), hash2.String())
}

func TestRecursivityOfAdd(t *testing.T) {
	t.Parallel()
	hash1 := common.NewRecursiveHash()
	hash1.Add([]byte("helloworld"))
	hash2 := common.NewRecursiveHash()
	hash2.Add([]byte("hello"))
	hash2.Add([]byte("world"))
	assert.NotEqual(t, hash1.String(), hash2.String())
}

func TestRecursivityOfFields(t *testing.T) {
	t.Parallel()
	hash1 := common.NewRecursiveHash(
		common.Field("hello", "world"),
	)
	hash2 := common.NewRecursiveHash(
		common.Field("hell", "o"),
		common.Field("worl", "d"),
	)
	assert.NotEqual(t, hash1.String(), hash2.String())
}
