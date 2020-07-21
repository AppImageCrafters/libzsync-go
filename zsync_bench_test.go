package zsync

import (
	"github.com/AppImageCrafters/libzsync-go/chunksmapper"
	"github.com/AppImageCrafters/libzsync-go/control"
	"github.com/AppImageCrafters/libzsync-go/sources"
	"github.com/stretchr/testify/assert"
	"io"
	"io/ioutil"
	"os"
	"testing"
)

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

			zsync := NewZSyncFromControl(zsyncControl)

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

	zsync := ZSync{
		BlockSize:      int64(zsyncControl.BlockSize),
		ChecksumsIndex: zsyncControl.ChecksumIndex,
		RemoteFileSize: zsyncControl.FileLength,
		RemoteFileUrl:  "https://github.com/AppImage/AppImageKit/releases/download/continuous/appimagetool-x86_64.AppImage",
	}

	output, err := os.Create("/tmp/appimagetool-new-x86_64.AppImage")
	assert.Nil(t, err)

	filePath := "/tmp/appimagetool-x86_64.AppImage"

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
