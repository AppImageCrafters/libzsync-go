package hasedbuffer

import (
	"bytes"
	"github.com/glycerine/rbuf"
	"golang.org/x/crypto/md4"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHashedBuffer_Write(t *testing.T) {
	buf := NewHashedBuffer(4)
	md4Sum := md4.New()
	_, _ = buf.Write([]byte("1111"))

	assert.Equal(t, "c400ea01", buf.RollingSumHex())
	md4Sum.Write([]byte("1111"))
	assert.Equal(t, md4Sum.Sum(nil), buf.CheckSum())
	md4Sum.Reset()

	_, _ = buf.Write([]byte("2222"))

	assert.Equal(t, "c800f401", buf.RollingSumHex())

	md4Sum.Write([]byte("2222"))
	assert.Equal(t, md4Sum.Sum(nil), buf.CheckSum())
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

func TestHashedRingBufferZeoredTerminatedChunk(t *testing.T) {
	data := make([]byte, 2048)
	for i := 0; i < 60; i++ {
		data[i] = '2'
	}

	buf := NewHashedBuffer(2048)

	n, _ := buf.Write(data)
	assert.Equal(t, n, 2048)

	assert.Equal(t, buf.RollingSum(), []byte{184, 11, 76, 102})

	buf = NewHashedBuffer(2048)
	n, _ = buf.Write(data[:60])
	assert.Equal(t, n, 60)

	n, _ = buf.Write(data[60:])
	assert.Equal(t, n, 2048-60)

	assert.Equal(t, buf.RollingSum(), []byte{0, 0, 32, 255})
}

func TestHashedRingBufferChecksums(t *testing.T) {
	baseString := make([]byte, 2048*2+60)
	for i := range baseString {
		baseString[i] = []byte("0123456789")[(i/2048)%9]
	}

	expectedRollingSums := [][]byte{{0, 192}, {0, 196}, {76, 102}}
	expectedChecksums := [][]byte{{169, 65, 57}, {131, 128, 226}, {243, 188, 144}}

	buf := NewHashedBuffer(2048)
	zeroes := make([]byte, 2048)

	for i := 0; i < 3; i++ {
		bytesCopied, _ := io.CopyN(buf, bytes.NewReader(baseString[2048*i:]), 2048)
		if bytesCopied < 2048 {
			_, _ = io.CopyN(buf, bytes.NewReader(zeroes), 2048-bytesCopied)
		}

		rollSum := buf.RollingSum()[2:4]
		checkSum := buf.CheckSum()[0:3]

		assert.Equal(t, expectedRollingSums[i], rollSum)
		assert.Equal(t, expectedChecksums[i], checkSum)
	}

}
