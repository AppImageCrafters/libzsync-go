package chunks

import "sort"

type FileChunksMapper struct {
	fileSize  int64
	chunksMap map[int64]ChunkInfo
}

func NewFileChunksMapper(fileSize int64) *FileChunksMapper {
	return &FileChunksMapper{fileSize: fileSize, chunksMap: make(map[int64]ChunkInfo)}
}

func (mapper *FileChunksMapper) FillChunksMap(chunkChannel <-chan ChunkInfo) {
	for {
		chunk, ok := <-chunkChannel
		if ok == false {
			break
		}

		mapper.Add(chunk)
	}
}

func (mapper *FileChunksMapper) GetMappedChunks() []ChunkInfo {
	var chunkList []ChunkInfo
	for _, chk := range mapper.chunksMap {
		chunkList = append(chunkList, chk)
	}

	sort.SliceStable(chunkList, func(i, j int) bool {
		return chunkList[i].TargetOffset < chunkList[j].TargetOffset
	})

	return chunkList
}

func (mapper *FileChunksMapper) GetMissingChunks() []ChunkInfo {
	chunkList := mapper.GetMappedChunks()
	var missingChunkList []ChunkInfo

	pastChunkEnd := int64(0)
	for _, c := range chunkList {
		if pastChunkEnd != c.TargetOffset {
			missingChunkList = append(missingChunkList, ChunkInfo{
				Size:         c.TargetOffset - pastChunkEnd,
				SourceOffset: pastChunkEnd,
				TargetOffset: pastChunkEnd,
			})
		}

		pastChunkEnd = c.TargetOffset + c.Size
	}

	if pastChunkEnd != mapper.fileSize {
		missingChunkList = append(missingChunkList, ChunkInfo{
			Size:         mapper.fileSize - pastChunkEnd,
			SourceOffset: pastChunkEnd,
			TargetOffset: pastChunkEnd,
		})
	}

	return missingChunkList
}

func (mapper *FileChunksMapper) Add(chunk ChunkInfo) {
	mapper.chunksMap[chunk.TargetOffset] = chunk
}
