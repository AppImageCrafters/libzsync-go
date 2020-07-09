package rollinghash

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRollingHash(t *testing.T) {
	hash := RollingHash{
		a: 0,
		b: 0,
	}

	for i := 0; i < 4; i++ {
		hash.Append('1', uint16(4-i))
	}
	assert.Equal(t, hash, RollingHash{
		a: 196,
		b: 490,
	})

	expectedResults := []RollingHash{
		{a: 197, b: 59951},
		{a: 198, b: 53877},
		{a: 199, b: 47804},
		{a: 200, b: 41732},
	}

	for i := 0; i < 4; i++ {
		hash.Update('2', '1', 7)
		assert.Equal(t, hash, expectedResults[i])
	}
}
