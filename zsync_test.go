package zsync

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
	"time"

	"github.com/AppImageCrafters/libzsync-go/chunks"
	"github.com/AppImageCrafters/libzsync-go/control"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	teardown()
	os.Exit(code)
}

var dataDir string = "/tmp/appimage-update"
var serverUrl string = ""

func getControl(filepath string) (zsyncControl *control.Control, err error) {
	file, err := os.Open(dataDir + "/" + filepath)
	if err != nil {
		return nil, err
	}
	zsyncControl, err = control.ReadControl(file)
	if err != nil {
		return nil, err
	}

	return zsyncControl, nil
}

func teardown() {
	os.RemoveAll(dataDir)
}

func downloadFile(filepath string, url string) error {
	if _, err := os.Stat(filepath); err == nil {
		return nil
	}
	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}

func setup() {
	dataDir := generateTestDataDir()
	downloadFile("/tmp/appimagetool-x86_64.AppImage.zsync", "https://github.com/AppImage/AppImageKit/releases/download/continuous/appimagetool-x86_64.AppImage.zsync")
	downloadFile("/tmp/appimagetool-x86_64.AppImage", "https://github.com/AppImage/AppImageKit/releases/download/12/appimagetool-x86_64.AppImage")
	serve(dataDir)
}

func serve(dataDir string) {
	srv := &http.Server{Addr: ":8080"}
	serverUrl = "http://localhost:8080/"

	http.Handle("/", http.FileServer(http.Dir(dataDir)))
	go srv.ListenAndServe()
}

func generateTestDataDir() string {
	err := os.MkdirAll(dataDir, 0777)
	if err != nil {
		log.Fatal(err)
	}

	rand.Seed(time.Now().UnixNano())

	_ = GenerateSampleFile([]byte("0123456789"), 2048*2+60, 0, dataDir+"/file")
	_ = GenerateSampleFile([]byte("0123456789"), 2048*2+70, 1, dataDir+"/file_displaced")
	makeZsyncFile(dataDir+"/file", err)

	_ = GenerateSampleFile([]byte("x123456789"), 2048*2+60, 0, dataDir+"/1st_chunk_changed")
	_ = GenerateSampleFile([]byte("0x23456789"), 2048*2+60, 0, dataDir+"/2nd_chunk_changed")
	_ = GenerateSampleFile([]byte("01x3456789"), 2048*2+60, 0, dataDir+"/3rd_chunk_changed")
	_ = GenerateSampleFile([]byte("abc3456789"), 2048*2+60, 0, dataDir+"/all_changed")
	_ = GenerateSampleFile([]byte("abc3456789"), 2048*200+60, 0, dataDir+"/large_file")

	_ = GenerateSampleFile([]byte("0123456789"), 2048*runtime.NumCPU()+500, 0, dataDir+"/uneven_workload_complete")
	makeZsyncFile(dataDir+"/uneven_workload_complete", err)

	return dataDir
}

func GenerateSampleFile(chars []byte, size int, offset int, filePath string) (err error) {
	baseString := make([]byte, size)
	for i := range baseString {
		baseString[i] = chars[((offset+i)/2048)%len(chars)]
	}

	err = writeStringToFile(filePath, baseString)
	if err != nil {
		log.Fatal(err)
	}

	return nil
}

func writeStringToFile(baseFilePath string, baseString []byte) error {
	err := ioutil.WriteFile(baseFilePath, baseString, 0666)
	if err != nil {
		fmt.Print(err)
	}
	return err
}

func makeZsyncFile(baseFileName string, err error) string {
	cmd := exec.Command("zsyncmake", baseFileName)
	cmd.Dir = filepath.Dir(baseFileName)
	err = cmd.Run()
	if err != nil {
		fmt.Print(err)
	}

	return baseFileName + ".zsync"
}

func TestZSync2_Sync(t *testing.T) {
	tests := []string{
		"/file_displaced",
		"/1st_chunk_changed",
		"/2nd_chunk_changed",
		"/3rd_chunk_changed",
		"/all_changed",
	}

	for _, tt := range tests {
		t.Run(tt, func(t *testing.T) {
			zsyncControl, _ := getControl("file.zsync")
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

func TestZSync2_SearchReusableChunks(t *testing.T) {
	zsyncControl, _ := getControl("file.zsync")
	zsyncControl.URL = serverUrl + "file"

	zsync := NewZSyncFromControl(zsyncControl)

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

func TestZSync2_SearchReusableChunksWithSyncedFile(t *testing.T) {
	numCPU := runtime.NumCPU()

	zsyncControl, _ := getControl("uneven_workload_complete.zsync")
	zsyncControl.URL = serverUrl + "uneven_workload_complete"

	zsync := NewZSyncFromControl(zsyncControl)

	var results []chunks.ChunkInfo
	chunkChan, err := zsync.SearchReusableChunks(dataDir + "/uneven_workload_complete")
	assert.Nil(t, err)

	for chunk := range chunkChan {
		results = append(results, chunk)
	}
	sort.SliceStable(results, func(i, j int) bool {
		return results[i].TargetOffset < results[j].TargetOffset
	})

	assert.Len(t, results, numCPU+1)
	assert.Equal(t, int64(zsyncControl.BlockSize), results[0].Size)

	assert.Equal(t, int64(zsyncControl.BlockSize*uint(numCPU)), results[numCPU].SourceOffset)
	assert.Equal(t, int64(500), results[numCPU].Size)
}

func TestZSync2_WriteChunks(t *testing.T) {
	zsync := ZSync{
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

	sourceDataReader := bytes.NewReader(sourceData)
	for {
		chunk, ok := <-chunkChan
		if ok == false {
			break
		}

		err := zsync.WriteChunk(sourceDataReader, output, chunk)
		assert.Equal(t, err, nil)
	}

	// read result
	output.Seek(0, io.SeekStart)
	resultData, err := ioutil.ReadAll(output)
	assert.Equal(t, err, nil)

	expectedData := []byte{0, 1, 2}
	assert.Equal(t, resultData, expectedData)
}
