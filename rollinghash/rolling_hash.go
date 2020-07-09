package rollinghash

import "encoding/binary"

type RollingHash struct {
	a uint16
	b uint16
}

// Puts the sum into b. Avoids allocation. b must have length >= 4
func (r *RollingHash) PutSum(b []byte) {
	value := uint32(r.a) | uint32(r.b)<<16
	binary.LittleEndian.PutUint32(b, value)
}

func (r *RollingHash) Append(c byte, len uint16) {
	r.a += uint16(c)
	r.b += len * uint16(c)
}

func (r *RollingHash) Update(newC byte, oldC byte, shift int) {
	r.a += uint16(newC - oldC)
	r.b += r.a - uint16(oldC)<<shift
}
