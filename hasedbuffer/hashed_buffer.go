package hasedbuffer

import (
	"encoding/hex"
	"github.com/AppImageCrafters/libzsync-go/rollinghash"
	"github.com/glycerine/rbuf"
	"golang.org/x/crypto/md4"
	"io"
)

type HashedRingBuffer struct {
	hash *rollinghash.RollingHash
	rBuf *rbuf.FixedSizeRingBuf
}

func NewHashedBuffer(size int) *HashedRingBuffer {
	/* Calculate bit-shift for blocksize */
	var blockShift uint16
	for i := uint16(0); i < 32; i++ {
		if size <= (1 << i) {
			blockShift = i
			break
		}
	}

	return &HashedRingBuffer{
		hash: rollinghash.NewRollingHash(blockShift),
		rBuf: rbuf.NewFixedSizeRingBuf(size),
	}
}

func (h *HashedRingBuffer) Write(p []byte) (n int, err error) {
	pSize := len(p)
	evictedSize := (h.rBuf.Readable + pSize) - h.rBuf.N
	if evictedSize < 0 {
		evictedSize = 0
	}

	for i := 0; i < pSize; i++ {
		if i < evictedSize {
			evicted := uint16(h.rBuf.A[h.rBuf.Use][h.rBuf.Beg])
			h.hash.Update(uint16(p[i]), evicted)

			h.rBuf.Advance(1)
		} else {
			h.hash.Update(uint16(p[i]), 0)
		}
	}

	return h.rBuf.Write(p)
}

func (h HashedRingBuffer) Bytes() []byte {
	return h.rBuf.Bytes()
}

func (h HashedRingBuffer) RollingSumHex() string {
	sum := h.RollingSum()

	return hex.EncodeToString(sum)
}

func (h HashedRingBuffer) RollingSum() []byte {
	sum := make([]byte, 4)
	h.hash.PutSum(sum)
	return sum
}

func (h HashedRingBuffer) CheckSum() []byte {
	sumBuilder := md4.New()
	slice1, slice2 := h.rBuf.BytesTwo(false)
	sumBuilder.Write(slice1)
	sumBuilder.Write(slice2)
	sum := sumBuilder.Sum(nil)

	return sum
}

func (h HashedRingBuffer) CheckSumHex() string {
	sum := h.CheckSum()

	return hex.EncodeToString(sum)
}

func (h *HashedRingBuffer) ReadNFrom(input io.Reader, bytes int64) (int64, error) {
	newBytesIdx := h.rBuf.Last() + 1
	n, readErr := h.rBuf.ReadFrom(io.LimitReader(input, bytes))
	evictedSize := h.rBuf.Readable - h.rBuf.N
	if evictedSize < 0 {
		evictedSize = 0
	}

	for i := 0; i < int(n); i++ {
		newChar := uint16(h.rBuf.A[h.rBuf.Use][newBytesIdx])

		if i < evictedSize {
			evicted := uint16(h.rBuf.A[h.rBuf.Use][h.rBuf.Beg])
			h.hash.Update(newChar, evicted)

			h.rBuf.Advance(1)
		} else {
			h.hash.Update(newChar, 0)
		}

		newBytesIdx = h.rBuf.Nextpos(newBytesIdx)
	}

	return n, readErr
}

// Read the complete buffer from input, missing bytes are replaced by '0'
func (h *HashedRingBuffer) ReadFull(input io.Reader) (int64, error) {
	h.rBuf.Reset()
	h.hash.Reset()

	newCharIdx := h.rBuf.Beg + h.rBuf.Readable
	n, err := h.rBuf.ReadFrom(input)

	missingChars := uint16(h.rBuf.N) - uint16(n)
	for rsunLen := uint16(h.rBuf.N); rsunLen > missingChars; rsunLen-- {
		newChar := uint16(h.rBuf.A[h.rBuf.Use][newCharIdx])
		newCharIdx = h.rBuf.Nextpos(newCharIdx)

		h.hash.Append(newChar, rsunLen)
	}

	for ; missingChars > 0; missingChars-- {
		h.hash.Append(0, missingChars)
	}

	return n, err
}
