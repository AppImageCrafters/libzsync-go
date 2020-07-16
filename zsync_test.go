package zsync

import (
	"bytes"
	"fmt"
	"github.com/AppImageCrafters/zsync/chunks"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/AppImageCrafters/zsync/control"
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

func TestZSync2_SearchReusableChunks(t *testing.T) {
	zsyncControl, _ := getControl()
	zsyncControl.URL = serverUrl + "file"

	zsync := ZSync2{
		BlockSize:      int64(zsyncControl.BlockSize),
		checksumsIndex: zsyncControl.ChecksumIndex,
	}

	chunks, _ := zsync.SearchReusableChunks(dataDir + "/1st_chunk_changed")
	chunk, _ := <-chunks

	assert.Equal(t, chunk.Size, int64(zsyncControl.BlockSize))
	assert.Equal(t, chunk.SourceOffset, int64(zsyncControl.BlockSize))

	chunk, _ = <-chunks
	assert.Equal(t, chunk.Size, int64(60))
	assert.Equal(t, chunk.SourceOffset, int64(zsyncControl.BlockSize*2))

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

func TestSyncChunksDisplaced(t *testing.T) {
	zsyncControl, _ := getControl()
	zsyncControl.URL = serverUrl + "file"

	local, err := os.Open(dataDir + "/file_displaced")
	if err != nil {
		return
	}
	defer local.Close()

	output := bytes.Buffer{}

	err = Sync(local, &output, *zsyncControl, NewDummyProgressReporter())
	if err != nil {
		t.Fatal(err)
	}

	expected, _ := ioutil.ReadFile(dataDir + "/file")
	assert.Equal(t, expected, output.Bytes())
}

func TestSync1stChunkChanged(t *testing.T) {
	zsyncControl, _ := getControl()
	zsyncControl.URL = serverUrl + "file"

	local, err := os.Open(dataDir + "/1st_chunk_changed")
	if err != nil {
		return
	}
	defer local.Close()

	output := bytes.Buffer{}

	err = Sync(local, &output, *zsyncControl, NewDummyProgressReporter())
	if err != nil {
		t.Fatal(err)
	}

	expected, _ := ioutil.ReadFile(dataDir + "/file")
	assert.Equal(t, expected, output.Bytes())
}

func TestSync2ndChunkChanged(t *testing.T) {
	zsyncControl, _ := getControl()
	zsyncControl.URL = serverUrl + "file"

	local, err := os.Open(dataDir + "/2nd_chunk_changed")
	if err != nil {
		return
	}
	defer local.Close()

	output := bytes.Buffer{}

	err = Sync(local, &output, *zsyncControl, NewDummyProgressReporter())
	if err != nil {
		t.Fatal(err)
	}

	expected, _ := ioutil.ReadFile(dataDir + "/file")
	assert.Equal(t, expected, output.Bytes())
}

func TestSync3rdChunkChanged(t *testing.T) {
	zsyncControl, _ := getControl()
	zsyncControl.URL = serverUrl + "file"

	local, err := os.Open(dataDir + "/3rd_chunk_changed")
	if err != nil {
		return
	}
	defer local.Close()

	output := bytes.Buffer{}

	err = Sync(local, &output, *zsyncControl, NewDummyProgressReporter())
	if err != nil {
		t.Fatal(err)
	}

	expected, _ := ioutil.ReadFile(dataDir + "/file")
	assert.Equal(t, expected, output.Bytes())
}

func getControl() (zsyncControl *control.Control, err error) {
	data, err := ioutil.ReadFile(dataDir + "/file.zsync")
	if err != nil {
		return nil, err
	}
	zsyncControl, err = control.ParseControl(data)
	if err != nil {
		return nil, err
	}

	return zsyncControl, nil
}

func teardown() {
	os.RemoveAll(dataDir)
}

func setup() {
	dataDir := generateTestDataDir()
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
