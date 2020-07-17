package zsync

import (
	"fmt"
	"github.com/AppImageCrafters/zsync/sources"
	"io"
	"os"

	"github.com/AppImageCrafters/zsync/chunks"
	"github.com/AppImageCrafters/zsync/hasedbuffer"
	"github.com/AppImageCrafters/zsync/index"
)

type ZSync2 struct {
	BlockSize      int64
	checksumsIndex *index.ChecksumIndex

	RemoteFileUrl  string
	RemoteFileSize int64
}

func (zsync *ZSync2) Sync(filePath string, output io.WriteSeeker) error {
	reusableChunks, err := zsync.SearchReusableChunks(filePath)
	if err != nil {
		return err
	}

	input, err := os.Open(filePath)
	if err != nil {
		return err
	}

	chunkMapper := chunks.NewFileChunksMapper(zsync.RemoteFileSize)
	for chunk := range reusableChunks {
		err = zsync.WriteChunk(input, output, chunk)
		if err != nil {
			return err
		}

		chunkMapper.Add(chunk)
	}

	missingChunksSource := sources.HttpFileSource{URL: zsync.RemoteFileUrl, Size: zsync.RemoteFileSize}
	missingChunks := chunkMapper.GetMissingChunks()

	for _, chunk := range missingChunks {
		// fetch whole chunk to reduce the number of request
		_, err = missingChunksSource.Seek(chunk.SourceOffset, io.SeekStart)
		if err != nil {
			return err
		}

		err = missingChunksSource.Request(chunk.Size)
		if err != nil {
			return err
		}

		err = zsync.WriteChunk(&missingChunksSource, output, chunk)
		if err != nil {
			return err
		}
	}

	return nil
}

func (zsync *ZSync2) SearchReusableChunks(path string) (<-chan chunks.ChunkInfo, error) {
	input, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	inputSize, err := zsync.getFileSize(input)
	if err != nil {
		return nil, err
	}

	chunkChannel := make(chan chunks.ChunkInfo)
	go zsync.searchReusableChunksAsync(0, inputSize, input, chunkChannel)

	return chunkChannel, nil
}

func (zsync *ZSync2) getFileSize(input *os.File) (int64, error) {
	inputStat, err := input.Stat()
	if err != nil {
		return -1, err
	}

	return inputStat.Size(), nil
}

func (zsync *ZSync2) searchReusableChunksAsync(begin int64, end int64, input io.ReadCloser, chunkChannel chan chunks.ChunkInfo) {
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

	_ = input.Close()
	close(chunkChannel)
}

func (zsync *ZSync2) WriteChunks(source io.ReadSeeker, output io.WriteSeeker, chunkChannel <-chan chunks.ChunkInfo) error {
	for {
		chunk, ok := <-chunkChannel
		if ok == false {
			break
		}

		err := zsync.WriteChunk(source, output, chunk)
		if err != nil {
			return err
		}
	}

	return nil
}

func (zsync *ZSync2) WriteChunk(source io.ReadSeeker, target io.WriteSeeker, chunk chunks.ChunkInfo) error {
	_, err := source.Seek(chunk.SourceOffset, io.SeekStart)
	if err != nil {
		return fmt.Errorf("unable to seek source offset: %d", chunk.SourceOffset)
	}

	_, err = target.Seek(chunk.TargetOffset, io.SeekStart)
	if err != nil {
		return fmt.Errorf("unable to seek target offset: %d", chunk.TargetOffset)
	}

	n, err := io.CopyN(target, source, chunk.Size)
	if err != nil {
		return fmt.Errorf("unable to copy bytes: %d %s", n, err.Error())
	}

	return nil
}
