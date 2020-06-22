package zsync

import (
	"github.com/AppImageCrafters/zsync/chunks"
	"github.com/AppImageCrafters/zsync/circularbuffer"
	"hash"
	"io"
	"math"
)

type ChunkLookupSlice struct {
	chunkOffset         int64
	chunkSize           int64
	chunksSize          int64
	chunksCount         int64
	lastFullChunkOffset int64
	fileSize            int64
	file                io.ReadSeeker
	buffer              *circularbuffer.C2
	weakChecksum        hash.Hash
	strongChecksum      hash.Hash
}

func NewChunkLookupSlice(file io.ReadSeeker, chunkSize int64, weakChecksum hash.Hash, strongChecksum hash.Hash) (*ChunkLookupSlice, error) {
	fileSize, err := file.Seek(0, 2)
	if err != nil {
		return nil, err
	}
	_, err = file.Seek(0, 0)

	chunkCount := int64(math.Ceil(float64(fileSize) / float64(chunkSize)))

	lookupSlice := &ChunkLookupSlice{
		chunksSize:          chunkSize,
		chunksCount:         chunkCount,
		lastFullChunkOffset: (fileSize / chunkSize) * chunkSize,
		fileSize:            fileSize,
		file:                file,
		buffer:              circularbuffer.MakeC2Buffer(int(chunkSize)),
		weakChecksum:        weakChecksum,
		strongChecksum:      strongChecksum,
	}
	err = lookupSlice.readChunk()
	if err != nil {
		return nil, err
	}

	return lookupSlice, nil
}

func (s *ChunkLookupSlice) isEOF() bool {
	return s.chunkOffset > s.lastFullChunkOffset
}

func (s ChunkLookupSlice) getNextChunkSize() int64 {
	if s.chunkOffset+s.chunksSize > s.fileSize {
		return s.fileSize - s.chunkOffset
	} else {
		return s.chunksSize
	}
}

func (s *ChunkLookupSlice) consumeChunk() error {
	s.chunkOffset += s.chunksSize

	err := s.readChunk()
	return err
}

func (s *ChunkLookupSlice) consumeByte() error {
	s.chunkOffset += 1

	err := s.readByte()
	return err
}

func (s *ChunkLookupSlice) readByte() error {
	if s.chunkOffset+s.chunkSize > s.fileSize {
		s.chunkSize = s.fileSize - s.chunkOffset
	}

	_, err := io.CopyN(io.MultiWriter(s.buffer, s.weakChecksum), s.file, 1)
	if err == io.EOF {
		multiWriter := io.MultiWriter(s.buffer, s.weakChecksum)
		_, err = multiWriter.Write([]byte{1})
	}

	return nil
}

func (s *ChunkLookupSlice) readChunk() (err error) {
	n, err := io.CopyN(io.MultiWriter(s.buffer, s.weakChecksum), s.file, s.chunksSize)
	if err == io.EOF {
		zeroChunk := make([]byte, s.chunksSize-n)
		multiWriter := io.MultiWriter(s.buffer, s.weakChecksum)
		_, err = multiWriter.Write(zeroChunk)
	}

	s.chunkSize = n

	return
}

func (s *ChunkLookupSlice) getBlock() []byte {
	return s.buffer.GetBlock()
}

func (s ChunkLookupSlice) GetWeakChecksum() []byte {
	return s.weakChecksum.Sum(nil)
}

func (s ChunkLookupSlice) GetStrongChecksum() []byte {
	s.strongChecksum.Reset()
	s.strongChecksum.Write(s.getBlock())
	return s.strongChecksum.Sum(nil)
}

func (syncData *SyncData) SearchAllMatchingChunks() ([]chunks.ChunkInfo, error) {
	var matchingChunks []chunks.ChunkInfo
	lookupSlice, err := NewChunkLookupSlice(syncData.Local, int64(syncData.BlockSize),
		syncData.WeakChecksumBuilder,
		syncData.StrongChecksumBuilder)

	if err != nil {
		return nil, err
	}

	syncData.progress.SetDescription("Searching reusable chunks: ")
	syncData.progress.SetTotal(lookupSlice.fileSize)

	for !lookupSlice.isEOF() {
		syncData.progress.SetProgress(lookupSlice.chunkOffset)

		weakMatches := syncData.ChecksumIndex.FindWeakChecksum2(lookupSlice.GetWeakChecksum())
		if weakMatches != nil {
			strongSum := lookupSlice.GetStrongChecksum()
			strongMatches := syncData.ChecksumIndex.FindStrongChecksum2(strongSum, weakMatches)
			if strongMatches != nil {
				matchingChunks = syncData.appendMatchingChunks(matchingChunks, strongMatches, lookupSlice.chunkSize, lookupSlice.chunkOffset)
				err = lookupSlice.consumeChunk()
				continue
			}
		}

		err = lookupSlice.consumeByte()
		if err != nil {
			return nil, err
		}
	}

	syncData.progress.SetProgress(lookupSlice.fileSize)
	return matchingChunks, nil
}

func (syncData *SyncData) appendMatchingChunks(matchingChunks []chunks.ChunkInfo, matches []chunks.ChunkChecksum, chunkSize int64, offset int64) []chunks.ChunkInfo {
	for _, match := range matches {
		newChunk := chunks.ChunkInfo{
			Size:         chunkSize,
			Source:       syncData.Local,
			SourceOffset: offset,
			TargetOffset: int64(match.ChunkOffset * syncData.BlockSize),
		}

		// chop zero filled chunks at the end
		if newChunk.TargetOffset+newChunk.Size > syncData.FileLength {
			newChunk.Size = syncData.FileLength - newChunk.TargetOffset
		}
		matchingChunks = append(matchingChunks, newChunk)
	}
	return matchingChunks
}
