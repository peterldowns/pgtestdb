package common_test

import (
	"testing"

	"github.com/peterldowns/testy/assert"

	"github.com/peterldowns/testdb/migrators/common"
)

func TestRecursiveHash(t *testing.T) {
	t.Parallel()
	hash1, err := common.NewRecursiveHash()
	assert.Nil(t, err)
	hash2, err := common.NewRecursiveHash()
	assert.Nil(t, err)
	assert.Equal(t, hash1.String(), hash2.String())
}

func TestRecursivityOfAdd(t *testing.T) {
	t.Parallel()
	hash1, err := common.NewRecursiveHash()
	assert.Nil(t, err)
	assert.Nil(t, hash1.Add([]byte("helloworld")))
	hash2, err := common.NewRecursiveHash()
	assert.Nil(t, err)
	assert.Nil(t, hash2.Add([]byte("hello")))
	assert.Nil(t, hash2.Add([]byte("world")))
	assert.NotEqual(t, hash1.String(), hash2.String())
}

func TestRecursivityOfFields(t *testing.T) {
	t.Parallel()
	hash1, err := common.NewRecursiveHash(
		common.Field("hello", "world"),
	)
	assert.Nil(t, err)
	hash2, err := common.NewRecursiveHash(
		common.Field("hell", "o"),
		common.Field("worl", "d"),
	)
	assert.Nil(t, err)
	assert.NotEqual(t, hash1.String(), hash2.String())
}
