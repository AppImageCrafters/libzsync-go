package control

/**
Control provides parser for the zsync control (.zsync) files.
*/

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/AppImageCrafters/zsync/chunks"
	"github.com/AppImageCrafters/zsync/filechecksum"
	"github.com/AppImageCrafters/zsync/index"
)

type ControlHeaderHashLengths struct {
	ConsecutiveMatchNeeded uint
	WeakCheckSumBytes      uint
	StrongCheckSumBytes    uint
}

type ControlHeader struct {
	Version     string
	MTime       string
	FileName    string
	Blocks      uint
	BlockSize   uint
	FileLength  int64
	HashLengths ControlHeaderHashLengths
	URL         string
	SHA1        string
}

type Control struct {
	ControlHeader

	ChecksumIndex  *index.ChecksumIndex
	ChecksumLookup filechecksum.ChecksumLookup
}

func ParseControl(data []byte) (control *Control, err error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("Missing zsync control data")
	}
	header, dataStart, err := LoadControlHeader(data)
	if err != nil {
		return nil, err
	}

	control = &Control{header, nil, nil}
	control.ChecksumIndex, control.ChecksumLookup, header.Blocks, err = readChecksumIndex(data[dataStart:], header)

	return
}

func LoadControlHeader(data []byte) (header ControlHeader, dataStart int, err error) {
	slice := data[:]
	line_end := bytes.Index(slice, []byte("\n"))

	// the header end is marked by an empty line "\n"
	for line_end != 0 && line_end != -1 {
		dataStart += line_end + 1
		line := string(slice[:line_end])

		k, v := parseHeaderLine(line)
		setHeaderValue(&header, k, v)

		slice = slice[line_end+1:]
		line_end = bytes.Index(slice, []byte("\n"))
	}

	if line_end >= 0 {
		dataStart += line_end + 1
	}

	if header.BlockSize == 0 {
		return header, dataStart, fmt.Errorf("Malformed zsync control: missing BlockSize ")
	}

	header.Blocks = (uint(header.FileLength) + header.BlockSize - 1) / header.BlockSize

	return header, dataStart, nil
}

func setHeaderValue(header *ControlHeader, k string, v string) {
	switch k {
	case "zsync":
		header.Version = v
	case "filename":
		header.FileName = v
	case "mtime":
		header.MTime = v
	case "blocksize":
		vi, err := strconv.ParseUint(v, 10, 0)
		if err == nil {
			header.BlockSize = uint(vi)
		}

	case "length":
		vi, err := strconv.ParseInt(v, 10, 0)
		if err == nil {
			header.FileLength = vi
		}
	case "hash-lengths":
		hashLenghts, err := parseHashLengths(v)
		if err == nil {
			header.HashLengths = *hashLenghts
		}
	case "url":
		header.URL = v
	case "sha-1":
		header.SHA1 = v
	default:
		fmt.Println("Unknown zsync control key: " + k)
	}
}

func parseHashLengths(s string) (hashLengths *ControlHeaderHashLengths, err error) {
	const errorPrefix = "Invalid Hash-Lengths entry"
	parts := strings.Split(s, ",")
	hashLengthsArray := make([]uint, len(parts))

	for i, v := range parts {
		vi, err := strconv.ParseUint(v, 10, 0)
		if err == nil {
			hashLengthsArray[i] = uint(vi)
		} else {
			return nil, err
		}
	}

	if len(hashLengthsArray) != 3 {
		return nil,
			fmt.Errorf(errorPrefix + ", expected: " + " ConsecutiveMatchNeeded, WeakCheckSumBytes, StrongCheckSumBytes")
	}

	hashLengths = &ControlHeaderHashLengths{
		ConsecutiveMatchNeeded: hashLengthsArray[0],
		WeakCheckSumBytes:      hashLengthsArray[1],
		StrongCheckSumBytes:    hashLengthsArray[2],
	}

	if hashLengths.ConsecutiveMatchNeeded < 1 || hashLengths.ConsecutiveMatchNeeded > 2 {
		return nil, fmt.Errorf(errorPrefix + ": ConsecutiveMatchNeeded must be in rage [1, 2] ")
	}

	if hashLengths.WeakCheckSumBytes < 1 || hashLengths.WeakCheckSumBytes > 4 {
		return nil, fmt.Errorf(errorPrefix + ": WeakCheckSumBytes must be in rage [1, 4] ")
	}

	if hashLengths.StrongCheckSumBytes < 3 || hashLengths.StrongCheckSumBytes > 16 {
		return nil, fmt.Errorf(errorPrefix + ": StrongCheckSumBytes must be in rage [4, 16] ")
	}

	return hashLengths, nil
}

func parseHeaderLine(line string) (key string, value string) {
	parts := strings.SplitN(line, ":", 2)
	key = strings.ToLower(parts[0])

	if len(parts) == 2 {
		value = strings.TrimSpace(parts[1])
	}

	return key, value
}

func readChecksumIndex(dataSlice []byte, header ControlHeader) (i *index.ChecksumIndex,
	checksumLookup filechecksum.ChecksumLookup,
	blockCount uint,
	err error) {

	reader := bytes.NewReader(dataSlice)
	readChunks, err := chunks.LoadChecksumsFromReaderLegacy(
		reader,
		int(header.HashLengths.WeakCheckSumBytes),
		int(header.HashLengths.StrongCheckSumBytes),
	)

	if err != nil {
		return
	}

	checksumLookup = chunks.StrongChecksumGetter(readChunks)
	i = index.MakeChecksumIndex(readChunks,
		header.HashLengths.WeakCheckSumBytes,
		header.HashLengths.StrongCheckSumBytes)
	blockCount = uint(len(readChunks))
	return
}
