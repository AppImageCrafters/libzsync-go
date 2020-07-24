package rollinghash

import "encoding/binary"

type RollingHash struct {
	a     uint16
	b     uint16
	shift uint16
}

func NewRollingHash(shift uint16) *RollingHash {
	return &RollingHash{shift: shift}
}

// Puts the sum into b. Avoids allocation. b must have length >= 4
func (r *RollingHash) PutSum(b []byte) {
	value := uint32(r.a) | uint32(r.b)<<16
	binary.LittleEndian.PutUint32(b, value)
}

func (r *RollingHash) Append(c uint16, len uint16) {
	r.a += c
	r.b += len * c
}

func (r *RollingHash) Update(newC uint16, oldC uint16) {
	r.a += newC - oldC
	r.b += r.a - oldC<<r.shift
}

func (r *RollingHash) Reset() {
	r.a = 0
	r.b = 0
}
