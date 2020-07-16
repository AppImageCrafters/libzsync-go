package rollinghash

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRollingHash(t *testing.T) {
	hash := NewRollingHash(7)

	data := "1111"
	for _, c := range data {
		hash.Update(uint16(c), 0)
	}
	sum := make([]byte, 4)
	hash.PutSum(sum)
	assert.Equal(t, sum, []byte{196, 0, 234, 1})
}

func TestRollingHash_Update(t *testing.T) {
	rhash := NewRollingHash(7)
	sum := make([]byte, 4)

	rhash.PutSum(sum)
	assert.Equal(t, sum, []byte{0, 0, 0, 0})

	rhash.Update('1', 0)
	rhash.PutSum(sum)
	assert.Equal(t, sum, []byte{49, 0, 49, 0})

	rhash.Update('2', '1')
	rhash.PutSum(sum)
	assert.Equal(t, sum, []byte{50, 0, 227, 231})

	rhash.Update(0, '2')
	rhash.PutSum(sum)
	assert.Equal(t, sum, []byte{0, 0, 227, 206})
}
