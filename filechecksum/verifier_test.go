package filechecksum

import (
	"crypto/md5"
	"testing"
)

type SingleBlockSource []byte

func (d SingleBlockSource) GetStrongChecksumForBlock(blockID int) []byte {
	m := md5.New()
	m.Write(d)
	return m.Sum(nil)
}

func TestBlockEqualsItself(t *testing.T) {
	data := []byte("fooooo")
	if len(data) < int(16) {
		zeroFilledBlock := make([]byte, 16-uint(len(data)))
		data = append(data, zeroFilledBlock...)
	}

	h := HashVerifier{
		Hash:                md5.New(),
		BlockSize:           uint(len(data)),
		BlockChecksumGetter: SingleBlockSource(data),
		FinalChunkLen:       16,
	}

	if !h.VerifyBlockRange(0, data) {
		t.Error("data did not verify")
	}
}

type FourByteBlockSource []byte

func (d FourByteBlockSource) GetStrongChecksumForBlock(blockID int) []byte {
	m := md5.New()

	start := blockID * 4
	end := start + 4

	if end >= len(d) {
		end = len(d)
	}

	m.Write(d[start:end])
	return m.Sum(nil)
}

func TestSplitBlocksEqualThemselves(t *testing.T) {
	data := []byte("foooBaar")
	if len(data) < int(16) {
		zeroFilledBlock := make([]byte, 16-uint(len(data)))
		data = append(data, zeroFilledBlock...)
	}

	h := HashVerifier{
		Hash:                md5.New(),
		BlockSize:           uint(4),
		BlockChecksumGetter: FourByteBlockSource(data),
		FinalChunkLen:       16,
	}

	if !h.VerifyBlockRange(0, data) {
		t.Error("data did not verify")
	}
}

func TestPartialBlock(t *testing.T) {
	data := []byte("fo")

	h := HashVerifier{
		Hash:                md5.New(),
		BlockSize:           uint(2),
		BlockChecksumGetter: SingleBlockSource(data),
		FinalChunkLen:       16,
	}

	if !h.VerifyBlockRange(0, data) {
		t.Error("data did not verify")
	}
}
