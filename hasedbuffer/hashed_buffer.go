package hasedbuffer

import (
	"encoding/hex"
	"github.com/AppImageCrafters/zsync/rollinghash"
	"github.com/glycerine/rbuf"
	"golang.org/x/crypto/md4"
)

type HashedRingBuffer struct {
	hash rollinghash.RollingHash
	rBuf rbuf.FixedSizeRingBuf
}

func NewHashedBuffer(size int) *HashedRingBuffer {
	return &HashedRingBuffer{
		hash: rollinghash.RollingHash{},
		rBuf: *rbuf.NewFixedSizeRingBuf(size),
	}
}

func (h *HashedRingBuffer) Write(p []byte) (n int, err error) {
	pLen := len(p)
	evicted := make([]byte, pLen)
	_, _ = h.rBuf.Read(evicted)

	for i := 0; i < pLen; i++ {
		h.hash.Update(p[i], evicted[i], 7)
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
	sum := md4.New()
	return sum.Sum(h.rBuf.Bytes())
}

func (h HashedRingBuffer) CheckSumHex() string {
	sum := h.CheckSum()

	return hex.EncodeToString(sum)
}
