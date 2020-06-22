package control

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLoadControlHeader(t *testing.T) {
	data := []byte(`zsync: 0.6.2
Filename: Hello World-latest-x86_64.AppImage
MTime: Fri, 08 May 2020 17:36:00 +0000
Blocksize: 2048
Length: 4096
Hash-Lengths: 2,2,5
URL: Hello World-latest-x86_64.AppImage
SHA-1: da7a3ee0ebb42db73f96c67438ff38c21204f676

`)

	controlHeader, _, _ := LoadControlHeader(data)
	assert.Equal(t, "0.6.2", controlHeader.Version)
	assert.Equal(t, "Hello World-latest-x86_64.AppImage", controlHeader.FileName)
	assert.Equal(t, "Fri, 08 May 2020 17:36:00 +0000", controlHeader.MTime)
	assert.Equal(t, uint(2048), controlHeader.BlockSize)
	assert.Equal(t, int64(4096), controlHeader.FileLength)
	assert.Equal(t, uint(2), controlHeader.HashLengths.ConsecutiveMatchNeeded)
	assert.Equal(t, uint(2), controlHeader.HashLengths.WeakCheckSumBytes)
	assert.Equal(t, uint(5), controlHeader.HashLengths.StrongCheckSumBytes)
	assert.Equal(t, "Hello World-latest-x86_64.AppImage", controlHeader.URL)
	assert.Equal(t, "da7a3ee0ebb42db73f96c67438ff38c21204f676", controlHeader.SHA1)
}
