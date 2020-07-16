package zsync

import (
	"github.com/AppImageCrafters/zsync/hasedbuffer"
	"github.com/AppImageCrafters/zsync/index"
	"io"

	"github.com/AppImageCrafters/zsync/chunks"
)

type ZSync2 struct {
	BlockSize      int64
	checksumsIndex *index.ChecksumIndex
}

func (zsync *ZSync2) SearchReusableChunks(input io.ReadSeeker) (chan chunks.ChunkInfo, error) {
	inputSize, err := input.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, err
	}

	// reset file offset to 0
	_, _ = input.Seek(0, io.SeekStart)

	chunkChannel := make(chan chunks.ChunkInfo)
	go zsync.searchReusableChunksAsync(0, inputSize, input, chunkChannel)

	return chunkChannel, nil
}

func (zsync *ZSync2) searchReusableChunksAsync(begin int64, end int64, input io.ReadSeeker, chunkChannel chan chunks.ChunkInfo) {
	nextStep := zsync.BlockSize

	maxOffset := (end / zsync.BlockSize) * zsync.BlockSize

	buf := hasedbuffer.NewHashedBuffer(int(zsync.BlockSize))

	for i := begin; i <= maxOffset; i += nextStep {
		n, err := io.CopyN(buf, input, nextStep)
		if err == io.EOF {
			zeroBuff := make([]byte, nextStep-n)
			_, err = buf.Write(zeroBuff)
		}

		if err != nil {
			break
		}

		weakSum := buf.RollingSum()
		weakMatches := zsync.checksumsIndex.FindWeakChecksum2(weakSum)

		if weakMatches != nil {
			strongSum := buf.CheckSum()
			strongMatches := zsync.checksumsIndex.FindStrongChecksum2(strongSum, weakMatches)
			if strongMatches != nil {
				for _, match := range strongMatches {
					newChunk := chunks.ChunkInfo{
						Size:         zsync.BlockSize,
						Source:       input,
						SourceOffset: i,
						TargetOffset: int64(match.ChunkOffset) * zsync.BlockSize,
					}

					// chop zero filled chunks at the end
					if newChunk.TargetOffset+newChunk.Size > end {
						newChunk.Size = end - newChunk.TargetOffset
					}

					chunkChannel <- newChunk
				}

				// consume entire block
				nextStep = int64(zsync.BlockSize)
				continue
			}
		}

		// just consume 1 byte
		nextStep = 1
	}

	close(chunkChannel)
}
