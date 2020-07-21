package chunksmapper

import (
	"github.com/AppImageCrafters/zsync/chunks"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFileChunksMapper_GetMissingChunks(t *testing.T) {
	mapper := ChunksMapper{
		fileSize:  12,
		chunksMap: make(map[int64]chunks.ChunkInfo),
	}

	chunkList := []chunks.ChunkInfo{
		chunks.ChunkInfo{TargetOffset: 2, Size: 2},
		chunks.ChunkInfo{TargetOffset: 4, Size: 2},
		chunks.ChunkInfo{TargetOffset: 8, Size: 2},
	}

	for _, chunk := range chunkList {
		mapper.chunksMap[chunk.TargetOffset] = chunk
	}

	result := mapper.GetMissingChunks()
	expected := []chunks.ChunkInfo{
		{TargetOffset: 0, SourceOffset: 0, Size: 2},
		{TargetOffset: 6, SourceOffset: 6, Size: 2},
		{TargetOffset: 10, SourceOffset: 10, Size: 2},
	}
	assert.Equal(t, expected, result)
}
