package control

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestReadControl(t *testing.T) {
	data := []byte(`zsync: 0.6.2
Filename: file
MTime: Tue, 21 Jul 2020 17:03:30 +0000
Blocksize: 2048
Length: 4156
Hash-Lengths: 2,2,3
URL: /tmp/appimage-update/file
SHA-1: 580c4e0ce970f2f9f311dc782e54127b1fa612ea

`)
	data = append(data, []byte{0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 2, 2, 2, 2, 2}...)

	reader := bytes.NewReader(data)
	c, err := ReadControl(reader)
	assert.Nil(t, err)

	assert.Equal(t, "0.6.2", c.Version)
	assert.Equal(t, "file", c.FileName)
	assert.Equal(t, "Tue, 21 Jul 2020 17:03:30 +0000", c.MTime)
	assert.Equal(t, uint(2048), c.BlockSize)
	assert.Equal(t, int64(4156), c.FileLength)
	assert.Equal(t, uint(2), c.HashLengths.ConsecutiveMatchNeeded)
	assert.Equal(t, uint(2), c.HashLengths.WeakCheckSumBytes)
	assert.Equal(t, uint(3), c.HashLengths.StrongCheckSumBytes)
	assert.Equal(t, "/tmp/appimage-update/file", c.URL)
	assert.Equal(t, "580c4e0ce970f2f9f311dc782e54127b1fa612ea", c.SHA1)

	assert.Equal(t, uint(3), c.Blocks)
	assert.NotNil(t, c.ChecksumIndex.FindWeakChecksum2([]byte{0, 0, 0, 0}))
	assert.NotNil(t, c.ChecksumIndex.FindWeakChecksum2([]byte{0, 0, 1, 1}))
	assert.NotNil(t, c.ChecksumIndex.FindWeakChecksum2([]byte{0, 0, 2, 2}))
}
