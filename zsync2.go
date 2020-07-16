package zsync

import (
	"fmt"
	"io"
	"os"

	"github.com/AppImageCrafters/zsync/chunks"
	"github.com/AppImageCrafters/zsync/hasedbuffer"
	"github.com/AppImageCrafters/zsync/index"
)

type ZSync2 struct {
	BlockSize      int64
	checksumsIndex *index.ChecksumIndex
}

func (zsync *ZSync2) SearchReusableChunks(path string) (<-chan chunks.ChunkInfo, error) {
	input, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer input.Close()

	inputSize, err := zsync.getFileSize(input)
	if err != nil {
		return nil, err
	}

	chunkChannel := make(chan chunks.ChunkInfo)
	go zsync.searchReusableChunksAsync(0, inputSize, input, chunkChannel)

	return chunkChannel, nil
}

func (zsync *ZSync2) getFileSize(input *os.File) (int64, error) {
	inputSize, err := input.Seek(0, io.SeekEnd)
	if err != nil {
		return -1, err
	}

	// reset file offset to 0
	_, _ = input.Seek(0, io.SeekStart)
	return inputSize, nil
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
						Source:       nil,
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

func (zsync *ZSync2) WriteChunks(source io.ReadSeeker, output io.WriteSeeker, chunkChannel <-chan chunks.ChunkInfo) error {
	for {
		chunk, ok := <-chunkChannel
		if ok == false {
			break
		}

		_, err := source.Seek(chunk.SourceOffset, io.SeekStart)
		if err != nil {
			return fmt.Errorf("unable to seek source offset: %d", chunk.SourceOffset)
		}

		_, err = output.Seek(chunk.TargetOffset, io.SeekStart)
		if err != nil {
			return fmt.Errorf("unable to seek target offset: %d", chunk.TargetOffset)
		}

		_, err = io.CopyN(output, source, chunk.Size)
		if err != nil {
			return fmt.Errorf("unable to copy bytes: %s", err.Error())
		}
	}

	return nil
}
