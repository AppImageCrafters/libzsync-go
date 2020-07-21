package chunksmapper

import (
	"github.com/AppImageCrafters/zsync/chunks"
	"sort"
)

type ChunksMapper struct {
	fileSize  int64
	chunksMap map[int64]chunks.ChunkInfo
}

func NewFileChunksMapper(fileSize int64) *ChunksMapper {
	return &ChunksMapper{fileSize: fileSize, chunksMap: make(map[int64]chunks.ChunkInfo)}
}

func (mapper *ChunksMapper) FillChunksMap(chunkChannel <-chan chunks.ChunkInfo) {
	for {
		chunk, ok := <-chunkChannel
		if ok == false {
			break
		}

		mapper.Add(chunk)
	}
}

func (mapper *ChunksMapper) GetMappedChunks() []chunks.ChunkInfo {
	var chunkList []chunks.ChunkInfo
	for _, chk := range mapper.chunksMap {
		chunkList = append(chunkList, chk)
	}

	sort.SliceStable(chunkList, func(i, j int) bool {
		return chunkList[i].TargetOffset < chunkList[j].TargetOffset
	})

	return chunkList
}

func (mapper *ChunksMapper) GetMissingChunks() []chunks.ChunkInfo {
	chunkList := mapper.GetMappedChunks()
	var missingChunkList []chunks.ChunkInfo

	pastChunkEnd := int64(0)
	for _, c := range chunkList {
		if pastChunkEnd != c.TargetOffset {
			missingChunkList = append(missingChunkList, chunks.ChunkInfo{
				Size:         c.TargetOffset - pastChunkEnd,
				SourceOffset: pastChunkEnd,
				TargetOffset: pastChunkEnd,
			})
		}

		pastChunkEnd = c.TargetOffset + c.Size
	}

	if pastChunkEnd != mapper.fileSize {
		missingChunkList = append(missingChunkList, chunks.ChunkInfo{
			Size:         mapper.fileSize - pastChunkEnd,
			SourceOffset: pastChunkEnd,
			TargetOffset: pastChunkEnd,
		})
	}

	return missingChunkList
}

func (mapper *ChunksMapper) Add(chunk chunks.ChunkInfo) {
	mapper.chunksMap[chunk.TargetOffset] = chunk
}
