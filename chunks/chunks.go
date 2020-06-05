/*
Package chunks provides the basic structure for a pair of the weak and strong checksums.
Since this is fairly widely used, splitting this out breaks a number of possible circular dependencies
*/
package chunks

import (
	"bytes"
	"errors"
	"io"
)

// For a given Block, the Weak & Strong hashes, and the offset
// This structure is only used to generate the index of reference files, since
// computing the strong checksum is not done when comparing unless the weak checksum matches
type ChunkChecksum struct {
	// an offset in terms of chunk count
	ChunkOffset uint
	// the size of the block
	Size           int64
	WeakChecksum   []byte
	StrongChecksum []byte
}

// compares a checksum to another based on the checksums, not the offset
func (chunk ChunkChecksum) Match(other ChunkChecksum) bool {
	weakEqual := bytes.Compare(chunk.WeakChecksum, other.WeakChecksum) == 0
	strongEqual := false
	if weakEqual {
		strongEqual = bytes.Compare(chunk.StrongChecksum, other.StrongChecksum) == 0
	}
	return weakEqual && strongEqual
}

var ErrPartialChecksum = errors.New("Reader length was not a multiple of the checksums")

// Loads chunks from a sources, assuming alternating weak then strong hashes
// This function attempts to be compatible with the original zsync implementation therefore the following assumptions
// are made:
// - weak checksums are stored in Little Endian notation
// - only first <strongHashSize> bytes of the strong checksum are provided
func LoadChecksumsFromReaderLegacy(
	r io.Reader,
	weakHashSize int,
	strongHashSize int,
) ([]ChunkChecksum, error) {

	result := make([]ChunkChecksum, 0, 20)
	offset := uint(0)

	temp := ChunkChecksum{}

	for {
		weakBuffer := make([]byte, weakHashSize)
		n, err := io.ReadFull(r, weakBuffer)

		if n == weakHashSize {
			temp.ChunkOffset = offset
			temp.WeakChecksum = TransformToInternalRepresentation(weakBuffer)
		} else if n == 0 && err == io.EOF {
			break
		} else {
			return nil, ErrPartialChecksum
		}

		strongBuffer := make([]byte, strongHashSize)
		n, err = io.ReadFull(r, strongBuffer)

		if n == strongHashSize {
			temp.StrongChecksum = strongBuffer
			result = append(result, temp)

			if err == io.EOF {
				break
			}
		} else {
			return nil, ErrPartialChecksum
		}

		offset += 1
	}

	return result, nil
}

// Required for zsync legacy support
func TransformToInternalRepresentation(inWeakBuffer []byte) []byte {
	weakBuffer := make([]byte, 4)

	// reverse bytes order
	for i, c := range inWeakBuffer {
		weakBuffer[3-i] = c
	}

	return weakBuffer
}

// satisfies filechecksum.ChecksumLookup
type StrongChecksumGetter []ChunkChecksum

func (s StrongChecksumGetter) GetStrongChecksumForBlock(blockID int) []byte {
	return s[blockID].StrongChecksum
}

type ChunkInfo struct {
	Size         int64
	Source       io.ReadSeeker
	SourceOffset int64
	TargetOffset int64
}
