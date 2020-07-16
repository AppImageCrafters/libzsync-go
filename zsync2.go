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

func (zsync *ZSync2) SearchReusableChunks(seed io.ReadSeeker) (chan chunks.ChunkInfo, error) {
	seedSize, err := seed.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, err
	}

	// reset file idx
	_, _ = seed.Seek(0, io.SeekStart)

	chunkChannel := make(chan chunks.ChunkInfo)
	go func() {

		nextStep := zsync.BlockSize

		maxOffset := (seedSize / zsync.BlockSize) * zsync.BlockSize

		buf := hasedbuffer.NewHashedBuffer(int(zsync.BlockSize))

		for i := int64(0); i <= maxOffset; i += nextStep {
			n, err := io.CopyN(buf, seed, nextStep)
			if err == io.EOF {
				zeroBuff := make([]byte, nextStep-n)
				_, err = buf.Write(zeroBuff)
			}

			if err != nil {
				return
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
							Source:       seed,
							SourceOffset: i,
							TargetOffset: int64(match.ChunkOffset) * zsync.BlockSize,
						}

						// chop zero filled chunks at the end
						if newChunk.TargetOffset+newChunk.Size > seedSize {
							newChunk.Size = seedSize - newChunk.TargetOffset
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
	}()

	return chunkChannel, nil
}
