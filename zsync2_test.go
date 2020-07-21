package zsync

import (
	"bytes"
	"github.com/AppImageCrafters/zsync/chunksmapper"
	"io"
	"io/ioutil"
	"os"
	"sort"
	"testing"

	"github.com/AppImageCrafters/zsync/chunks"
	"github.com/AppImageCrafters/zsync/control"
	"github.com/AppImageCrafters/zsync/sources"
	"github.com/stretchr/testify/assert"
)

func TestZSync2_Sync(t *testing.T) {
	tests := []string{
		"/file_displaced",
		"/1st_chunk_changed",
		"/2nd_chunk_changed",
		"/3rd_chunk_changed",
	}

	for _, tt := range tests {
		t.Run(tt, func(t *testing.T) {
			zsyncControl, _ := getControl()
			zsyncControl.URL = serverUrl + "file"

			zsync := ZSync2{
				BlockSize:      int64(zsyncControl.BlockSize),
				ChecksumsIndex: zsyncControl.ChecksumIndex,
				RemoteFileUrl:  zsyncControl.URL,
				RemoteFileSize: zsyncControl.FileLength,
			}

			outputPath := dataDir + "/file_copy"
			output, err := os.Create(outputPath)
			assert.Equal(t, err, nil)
			defer output.Close()

			err = zsync.Sync(dataDir+tt, output)
			if err != nil {
				t.Fatal(err)
			}

			expected, _ := ioutil.ReadFile(dataDir + "/file")
			result, _ := ioutil.ReadFile(outputPath)
			assert.Equal(t, expected, result)

			_ = os.Remove(outputPath)
		})
	}
}

func TestZSync2_SearchReusableChunks(t *testing.T) {
	zsyncControl, _ := getControl()
	zsyncControl.URL = serverUrl + "file"

	zsync := ZSync2{
		BlockSize:      int64(zsyncControl.BlockSize),
		ChecksumsIndex: zsyncControl.ChecksumIndex,
		RemoteFileSize: zsyncControl.FileLength,
	}

	var results []chunks.ChunkInfo
	chunkChan, err := zsync.SearchReusableChunks(dataDir + "/1st_chunk_changed")
	assert.Nil(t, err)

	for chunk := range chunkChan {
		results = append(results, chunk)
	}
	sort.SliceStable(results, func(i, j int) bool {
		return results[i].TargetOffset < results[j].TargetOffset
	})

	if len(results) != 2 {
		t.Fatalf("2 chunks expected")
	} else {
		assert.Equal(t, int64(zsyncControl.BlockSize), results[0].Size)
		assert.Equal(t, int64(zsyncControl.BlockSize), results[0].SourceOffset)

		assert.Equal(t, int64(60), results[1].Size)
		assert.Equal(t, int64(zsyncControl.BlockSize*2), results[1].SourceOffset)
	}
}

func TestZSync2_WriteChunks(t *testing.T) {
	zsync := ZSync2{
		BlockSize:      2,
		ChecksumsIndex: nil,
	}

	chunkChan := make(chan chunks.ChunkInfo)

	sourceData := []byte{1, 2}
	output, err := os.Create(dataDir + "/file_copy")
	assert.Equal(t, err, nil)
	defer output.Close()

	go func() {
		chunkChan <- chunks.ChunkInfo{TargetOffset: 1, Size: 2}
		close(chunkChan)
	}()

	err = zsync.WriteChunks(bytes.NewReader(sourceData), output, chunkChan)
	assert.Equal(t, err, nil)

	// read result
	output.Seek(0, io.SeekStart)
	resultData, err := ioutil.ReadAll(output)
	assert.Equal(t, err, nil)

	expectedData := []byte{0, 1, 2}
	assert.Equal(t, resultData, expectedData)
}

func BenchmarkZSync2_Sync(t *testing.B) {
	tests := []string{
		"/file_displaced",
		"/1st_chunk_changed",
		"/2nd_chunk_changed",
		"/3rd_chunk_changed",
	}

	for _, tt := range tests {
		t.Run(tt, func(t *testing.B) {
			zsyncControl, _ := getControl()
			zsyncControl.URL = serverUrl + "file"

			zsync := ZSync2{
				BlockSize:      int64(zsyncControl.BlockSize),
				ChecksumsIndex: zsyncControl.ChecksumIndex,
				RemoteFileUrl:  zsyncControl.URL,
				RemoteFileSize: zsyncControl.FileLength,
			}

			outputPath := dataDir + "/file_copy"
			output, err := os.Create(outputPath)
			assert.Equal(t, err, nil)
			defer output.Close()

			err = zsync.Sync(dataDir+tt, output)
			if err != nil {
				t.Fatal(err)
			}

			expected, _ := ioutil.ReadFile(dataDir + "/file")
			result, _ := ioutil.ReadFile(outputPath)
			assert.Equal(t, expected, result)

			_ = os.Remove(outputPath)
		})
	}
}

func BenchmarkZSync2_SyncAppImageTool(t *testing.B) {
	data, err := ioutil.ReadFile("/tmp/appimagetool-x86_64.AppImage.zsync")
	assert.Nil(t, err)

	zsyncControl, err := control.ParseControl(data)
	assert.Nil(t, err)

	zsync := ZSync2{
		BlockSize:      int64(zsyncControl.BlockSize),
		ChecksumsIndex: zsyncControl.ChecksumIndex,
		RemoteFileSize: zsyncControl.FileLength,
		RemoteFileUrl:  "https://github.com/AppImage/AppImageKit/releases/download/continuous/appimagetool-x86_64.AppImage",
	}

	output, err := os.Create("/tmp/appimagetool-new-x86_64.AppImage")
	assert.Nil(t, err)

	filePath := "/home/alexis/Downloads/appimagetool-x86_64.AppImage"

	reusableChunks, err := zsync.SearchReusableChunks(filePath)
	assert.Nil(t, err)

	input, err := os.Open(filePath)
	assert.Nil(t, err)

	chunkMapper := chunksmapper.NewFileChunksMapper(zsync.RemoteFileSize)
	for chunk := range reusableChunks {
		err = zsync.WriteChunk(input, output, chunk)
		assert.Nil(t, err)

		chunkMapper.Add(chunk)
	}

	missingChunksSource := sources.HttpFileSource{URL: zsync.RemoteFileUrl, Size: zsync.RemoteFileSize}
	missingChunks := chunkMapper.GetMissingChunks()

	for _, chunk := range missingChunks {
		// fetch whole chunk to reduce the number of request
		_, err = missingChunksSource.Seek(chunk.SourceOffset, io.SeekStart)
		assert.Nil(t, err)

		err = missingChunksSource.Request(chunk.Size)
		assert.Nil(t, err)

		err = zsync.WriteChunk(&missingChunksSource, output, chunk)
		assert.Nil(t, err)
	}

	assert.Nil(t, err)
}
