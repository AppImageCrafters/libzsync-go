package chunks

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFileChunksMapper_GetMissingChunks(t *testing.T) {
	mapper := FileChunksMapper{
		FileSize:     12,
		MappedChunks: make(map[int64]ChunkInfo),
	}

	chunkList := []ChunkInfo{
		ChunkInfo{TargetOffset: 2, Size: 2},
		ChunkInfo{TargetOffset: 4, Size: 2},
		ChunkInfo{TargetOffset: 8, Size: 2},
	}

	for _, chunk := range chunkList {
		mapper.MappedChunks[chunk.TargetOffset] = chunk
	}

	result := mapper.GetMissingChunks()
	expected := []ChunkInfo{
		{TargetOffset: 0, SourceOffset: 0, Size: 2},
		{TargetOffset: 6, SourceOffset: 6, Size: 2},
		{TargetOffset: 10, SourceOffset: 10, Size: 2},
	}
	assert.Equal(t, expected, result)
}
