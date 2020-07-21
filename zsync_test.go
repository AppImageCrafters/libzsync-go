package zsync

import (
	"fmt"
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
)

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	teardown()
	os.Exit(code)
}

var dataDir string = "/tmp/appimage-update"
var serverUrl string = ""

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
	downloadFile("/tmp/appimagetool-new-x86_64.AppImage", "https://github.com/AppImage/AppImageKit/releases/download/12/appimagetool-x86_64.AppImage")
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
