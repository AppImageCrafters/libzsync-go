package circularbuffer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHashedBuffer_Write(t *testing.T) {
	buf := NewHashedBuffer(90)
	// Fill
	_, _ = buf.Write([]byte("For control over HTTP client headers, redirect policy, and other settings, create a Client"))
	assert.Equal(t, buf.RollingHash, RollingHash{8319, 50761})

	// Updated
	_, _ = buf.Write([]byte("d"))
	assert.Equal(t, buf.RollingHash, RollingHash{8349, 50150})
}
