package filechecksum

import (
	"bytes"
	"chunks"
	"index"
	"os"
	"testing"
)

func TestChecksumGenerationEndsWithFilechecksum(t *testing.T) {
	const BLOCKSIZE = 100
	const BLOCK_COUNT = 20
	emptybuffer := bytes.NewBuffer(make([]byte, BLOCK_COUNT*BLOCKSIZE))

	checksum := NewFileChecksumGenerator(BLOCKSIZE)

	lastResult := ChecksumResults{}

	for lastResult = range checksum.StartChecksumGeneration(emptybuffer, 10, nil) {
	}

	if lastResult.Checksums != nil {
		t.Errorf("Last result had checksums: %#v", lastResult)
	}

	if lastResult.Filechecksum == nil {
		t.Errorf("Last result did not contain the filechecksum: %#v", lastResult)
	}
}

func TestChecksumGenerationReturnsCorrectChecksumCount(t *testing.T) {
	const BLOCKSIZE = 100
	const BLOCK_COUNT = 20

	emptybuffer := bytes.NewBuffer(make([]byte, BLOCK_COUNT*BLOCKSIZE))

	checksum := NewFileChecksumGenerator(BLOCKSIZE)

	resultCount := 0

	for r := range checksum.StartChecksumGeneration(emptybuffer, 10, nil) {
		resultCount += len(r.Checksums)
	}

	if resultCount != BLOCK_COUNT {
		t.Errorf("Unexpected block count returned: %v", resultCount)
	}
}

func TestChecksumGenerationContainsHashes(t *testing.T) {
	const BLOCKSIZE = 100
	const BLOCK_COUNT = 20

	emptybuffer := bytes.NewBuffer(make([]byte, BLOCK_COUNT*BLOCKSIZE))
	checksum := NewFileChecksumGenerator(BLOCKSIZE)

	for r := range checksum.StartChecksumGeneration(emptybuffer, 10, nil) {
		for _, r2 := range r.Checksums {
			if len(r2.WeakChecksum) != checksum.WeakRollingHash.Size() {
				t.Fatalf(
					"Wrong length weak checksum: %v vs %v",
					len(r2.WeakChecksum),
					checksum.WeakRollingHash.Size(),
				)
			}

			if len(r2.StrongChecksum) != checksum.StrongHash.Size() {
				t.Fatalf(
					"Wrong length strong checksum: %v vs %v",
					len(r2.StrongChecksum),
					checksum.StrongHash.Size(),
				)
			}
		}
	}
}

func TestRollsumLength(t *testing.T) {
	const BLOCKSIZE = 100
	const BLOCK_COUNT = 20

	emptybuffer := bytes.NewBuffer(make([]byte, BLOCK_COUNT*BLOCKSIZE))
	output := bytes.NewBuffer(nil)

	checksum := NewFileChecksumGenerator(BLOCKSIZE)

	// output length is expected to be 20 blocks
	expectedLength := (BLOCK_COUNT * checksum.GetStrongHash().Size()) +
		(BLOCK_COUNT * checksum.WeakRollingHash.Size())

	_, err := checksum.GenerateChecksums(emptybuffer, output)

	if err != nil {
		t.Fatal(err)
	}

	if output.Len() != expectedLength {
		t.Errorf(
			"output length (%v) did not match expected length (%v)",
			output.Len(),
			expectedLength,
		)
	}
}

func TestRollsumLengthWithPartialBlockAtEnd(t *testing.T) {
	const BLOCKSIZE = 100
	const FULL_BLOCK_COUNT = 20
	const BLOCK_COUNT = FULL_BLOCK_COUNT + 1

	emptybuffer := bytes.NewBuffer(make([]byte, FULL_BLOCK_COUNT*BLOCKSIZE+50))
	output := bytes.NewBuffer(nil)

	checksum := NewFileChecksumGenerator(BLOCKSIZE)

	// output length is expected to be 20 blocks
	expectedLength := (BLOCK_COUNT * checksum.GetStrongHash().Size()) +
		(BLOCK_COUNT * checksum.WeakRollingHash.Size())

	_, err := checksum.GenerateChecksums(emptybuffer, output)

	if err != nil {
		t.Fatal(err)
	}

	if output.Len() != expectedLength {
		t.Errorf(
			"output length (%v) did not match expected length (%v)",
			output.Len(),
			expectedLength,
		)
	}
}

func ExampleFileChecksumGenerator_LoadChecksumsFromReader() {
	const BLOCKSIZE = 8096
	checksum := NewFileChecksumGenerator(BLOCKSIZE)

	// This could be any sources that conforms to io.Reader
	// sections of a file, or the body of an http response
	file1, err := os.Open("fileChecksums.chk")

	if err != nil {
		return
	}

	defer file1.Close()

	ws, ss := checksum.GetChecksumSizes()
	checksums, err := chunks.LoadChecksumsFromReaderLegacy(file1, ws, ss)

	if err != nil {
		return
	}

	// Make an index that we can use against our local
	// checksums
	i := index.MakeChecksumIndex(checksums, 4, 16)

	// example checksum from a local file
	// look for the chunk in the index
	i.FindWeakChecksumInIndex([]byte("a"))

}
