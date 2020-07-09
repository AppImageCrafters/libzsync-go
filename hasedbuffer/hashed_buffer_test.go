package hasedbuffer

import (
	"github.com/glycerine/rbuf"
	"golang.org/x/crypto/md4"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHashedBuffer_Write(t *testing.T) {
	buf := NewHashedBuffer(4)
	md4Sum := md4.New()
	_, _ = buf.Write([]byte("1111"))

	assert.Equal(t, "c400ea01", buf.RollingSumHex())
	assert.Equal(t, md4Sum.Sum([]byte("1111")), buf.CheckSum())

	_, _ = buf.Write([]byte("2222"))

	assert.Equal(t, "c80004a3", buf.RollingSumHex())
	assert.Equal(t, md4Sum.Sum([]byte("2222")), buf.CheckSum())
}

func TestRbuf(t *testing.T) {
	buf := rbuf.NewFixedSizeRingBuf(4)

	_, _ = buf.Write([]byte("1111"))
	assert.Equal(t, buf.Bytes(), []byte("1111"))

	evictedBytes := make([]byte, 4)
	_, _ = buf.Read(evictedBytes)
	assert.Equal(t, evictedBytes, []byte("1111"))

	_, _ = buf.Write([]byte("2222"))
	assert.Equal(t, buf.Bytes(), []byte("2222"))
}
