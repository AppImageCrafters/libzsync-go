package circularbuffer

import (
	"encoding/binary"
)

type HashedBuffer struct {
	RollingHash

	Size   uint16
	hIdx   uint16
	len    uint16
	buffer []byte

	blockShift int
}

func NewHashedBuffer(size uint16) *HashedBuffer {
	blockShift := 0
	for ; blockShift < 32; blockShift++ {
		if size <= (1 << blockShift) {
			break
		}
	}

	return &HashedBuffer{
		Size:       size,
		blockShift: blockShift,
		buffer:     make([]byte, ^uint16(0)),
	}

}

func (h *HashedBuffer) Write(p []byte) (n int, err error) {
	var c byte
	i := -1

	for i, c = range p {
		h.buffer[h.hIdx] = c

		if h.len < h.Size {
			h.a += uint16(c)
			h.b += (h.Size - h.len) * uint16(c)

			h.len++
		} else {
			oldc := h.buffer[h.hIdx-h.Size]
			h.a += uint16(c - oldc)
			h.b += uint16(h.a) - uint16(oldc)<<h.blockShift
		}

		h.hIdx++
	}

	return i + 1, nil
}

type RollingHash struct {
	a uint16
	b uint16
}

// Puts the sum into b. Avoids allocation. b must have length >= 4
func (r *RollingHash) GetSum(b []byte) {
	value := uint32(r.a) | uint32(r.b)<<16
	binary.LittleEndian.PutUint32(b, value)
}
