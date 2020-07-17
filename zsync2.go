package zsync

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"

	"github.com/AppImageCrafters/zsync/chunks"
	"github.com/AppImageCrafters/zsync/hasedbuffer"
	"github.com/AppImageCrafters/zsync/index"
	"github.com/AppImageCrafters/zsync/sources"
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
	inputSize, err := zsync.getFileSize(path)
	if err != nil {
		return nil, err
	}

	nChunks := inputSize / zsync.BlockSize
	if nChunks*zsync.BlockSize < inputSize {
		nChunks++
	}

	nWorkers := int64(runtime.NumCPU())
	if nWorkers > nChunks {
		nWorkers = nChunks
	}

	nChunksPerWorker := nChunks / nWorkers

	chunkChannel := make(chan chunks.ChunkInfo)
	var waitGroup sync.WaitGroup

	waitGroup.Add(int(nWorkers))

	for i := int64(0); i < nWorkers; i++ {
		begin := i * zsync.BlockSize

		end := begin + nChunksPerWorker*zsync.BlockSize
		if end > inputSize {
			end = inputSize
		}

		go zsync.searchReusableChunksAsync(path, begin, end, chunkChannel, &waitGroup)
	}

	go func() {
		waitGroup.Wait()
		close(chunkChannel)
	}()

	return chunkChannel, nil
}

func (zsync *ZSync2) getFileSize(filePath string) (int64, error) {
	inputStat, err := os.Stat(filePath)
	if err != nil {
		return -1, err
	}

	return inputStat.Size(), nil
}

func (zsync *ZSync2) searchReusableChunksAsync(path string, begin int64, end int64, chunksChan chan<- chunks.ChunkInfo, wg *sync.WaitGroup) {
	defer wg.Done()

	input, err := os.Open(path)
	if err != nil {
		return
	}

	_, err = input.Seek(begin, io.SeekStart)
	if err != nil {
		return
	}

	nextStep := zsync.BlockSize

	buf := hasedbuffer.NewHashedBuffer(int(zsync.BlockSize))

	for off := begin; off < end; off += nextStep {
		err := zsync.consumeBytes(buf, input, nextStep)
		if err != nil {
			break
		}

		weakSum := buf.RollingSum()
		weakMatches := zsync.checksumsIndex.FindWeakChecksum2(weakSum)

		if weakMatches != nil {
			strongSum := buf.CheckSum()
			strongMatches := zsync.checksumsIndex.FindStrongChecksum2(strongSum, weakMatches)
			if strongMatches != nil {
				zsync.createChunks(strongMatches, off, end, chunksChan)

				// consume entire block
				nextStep = zsync.BlockSize
				continue
			}
		}

		// just consume 1 byte
		nextStep = 1
	}

	_ = input.Close()
}

func (zsync *ZSync2) consumeBytes(buf *hasedbuffer.HashedRingBuffer, input *os.File, nBytes int64) error {
	n, err := buf.ReadNFrom(input, nBytes)

	// fill missing bytes with 0
	if n != nBytes {
		zeroBuff := make([]byte, nBytes-n)
		_, err = buf.Write(zeroBuff)
	}

	return err
}

func (zsync *ZSync2) createChunks(strongMatches []chunks.ChunkChecksum, offset int64, end int64, chunksChan chan<- chunks.ChunkInfo) {
	for _, match := range strongMatches {
		newChunk := chunks.ChunkInfo{
			Size:         zsync.BlockSize,
			Source:       nil,
			SourceOffset: offset,
			TargetOffset: int64(match.ChunkOffset) * zsync.BlockSize,
		}

		// chop zero filled chunks at the end
		if newChunk.TargetOffset+newChunk.Size > zsync.RemoteFileSize {
			newChunk.Size = zsync.RemoteFileSize - newChunk.TargetOffset
		}

		chunksChan <- newChunk
	}
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
