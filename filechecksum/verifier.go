package filechecksum

import (
	"bytes"
	"hash"
)

type ChecksumLookup interface {
	GetStrongChecksumForBlock(blockID int) []byte
}

type HashVerifier struct {
	BlockSize           uint
	Hash                hash.Hash
	BlockChecksumGetter ChecksumLookup
	FinalChunkLen       uint
}

func (v *HashVerifier) VerifyBlockRange(startBlockID uint, data []byte) bool {
	for i := 0; i*int(v.BlockSize) < len(data); i++ {
		start := i * int(v.BlockSize)
		end := start + int(v.BlockSize)

		if end > len(data) {
			end = len(data)
		}

		blockData := data[start:end]

		expectedChecksum := v.BlockChecksumGetter.GetStrongChecksumForBlock(
			int(startBlockID) + i,
		)

		if expectedChecksum == nil {
			return true
		}

		if len(blockData) < int(v.BlockSize) {
			zeroFilledBlock := make([]byte, v.BlockSize-uint(len(blockData)))
			blockData = append(blockData, zeroFilledBlock...)
		}

		v.Hash.Write(blockData)
		hashedData := v.Hash.Sum(nil)
		if len(hashedData) > int(v.FinalChunkLen) {
			hashedData = hashedData[:v.FinalChunkLen]
		}

		if bytes.Compare(expectedChecksum, hashedData) != 0 {
			return false
		}

		v.Hash.Reset()
	}

	return true
}
