package hasedbuffer

import (
	"bytes"
	"io"
	"testing"

	"github.com/glycerine/rbuf"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/md4"
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

	assert.Equal(t, []byte{184, 11, 76, 102}, buf.RollingSum())

	buf = NewHashedBuffer(2048)
	n, _ = buf.Write(data[:60])
	assert.Equal(t, 60, n)

	n, _ = buf.Write(data[60:])
	assert.Equal(t, 2048-60, n)

	assert.Equal(t, []byte{184, 11, 76, 102}, buf.RollingSum())
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

func TestHashedRingBuffer_ReadFull(t *testing.T) {
	data := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	dataReader := bytes.NewReader(data)

	const chunkSize = 6
	buf := NewHashedBuffer(chunkSize)

	n, err := buf.ReadFull(dataReader)
	assert.Nil(t, err)
	assert.Equal(t, int64(6), n)
	assert.Equal(t, []byte{0, 1, 2, 3, 4, 5}, buf.Bytes())
	assert.Equal(t, []byte{15, 0, 35, 0}, buf.RollingSum())

	n, err = buf.ReadFull(dataReader)
	assert.Nil(t, err)
	assert.Equal(t, int64(6), n)
	assert.Equal(t, []byte{6, 7, 8, 9, 10, 11}, buf.Bytes())
	assert.Equal(t, []byte{51, 0, 161, 0}, buf.RollingSum())

	n, err = buf.ReadFull(dataReader)
	assert.Nil(t, err)
	assert.Equal(t, int64(4), n)
	assert.Equal(t, []byte{12, 13, 14, 15, 0, 0}, buf.Bytes())
	assert.Equal(t, []byte{54, 0, 238, 0}, buf.RollingSum())
}

func TestHashedRingBuffer_ReadByte(t *testing.T) {
	data := []byte{0, 1, 2}
	dataReader := bytes.NewReader(data)

	const chunkSize = 2
	buf := NewHashedBuffer(chunkSize)

	n, err := buf.ReadFull(dataReader)
	assert.Nil(t, err)
	assert.Equal(t, int64(2), n)
	assert.Equal(t, []byte{0, 1}, buf.Bytes())
	assert.Equal(t, []byte{1, 0, 1, 0}, buf.RollingSum())

	_, err = buf.ReadByte(dataReader)
	assert.Nil(t, err)
	assert.Equal(t, []byte{1, 2}, buf.Bytes())
	assert.Equal(t, []byte{3, 0, 4, 0}, buf.RollingSum())

	_, err = buf.ReadByte(dataReader)
	assert.Nil(t, err)
	assert.Equal(t, []byte{2}, buf.Bytes())
	assert.Equal(t, []byte{2, 0, 4, 0}, buf.RollingSum())
}
