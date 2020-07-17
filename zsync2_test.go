package zsync

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"testing"

	"github.com/AppImageCrafters/zsync/chunks"
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
				checksumsIndex: zsyncControl.ChecksumIndex,
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
		checksumsIndex: zsyncControl.ChecksumIndex,
	}

	chunks, _ := zsync.SearchReusableChunks(dataDir + "/1st_chunk_changed")
	chunk, _ := <-chunks

	assert.Equal(t, int64(zsyncControl.BlockSize), chunk.Size)
	assert.Equal(t, int64(zsyncControl.BlockSize), chunk.SourceOffset)

	chunk, _ = <-chunks
	assert.Equal(t, int64(60), chunk.Size)
	assert.Equal(t, int64(zsyncControl.BlockSize*2), chunk.SourceOffset)

	_, ok := <-chunks
	assert.False(t, ok)
}

func TestZSync2_WriteChunks(t *testing.T) {
	zsync := ZSync2{
		BlockSize:      2,
		checksumsIndex: nil,
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
