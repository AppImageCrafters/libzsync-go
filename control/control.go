package control

/**
Control provides parser for the zsync control (.zsync) files.
*/

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/AppImageCrafters/libzsync-go/chunks"
	"github.com/AppImageCrafters/libzsync-go/index"
)

type ControlHeaderHashLengths struct {
	ConsecutiveMatchNeeded uint
	WeakCheckSumBytes      uint
	StrongCheckSumBytes    uint
}

type Control struct {
	Version     string
	MTime       string
	FileName    string
	Blocks      uint
	BlockSize   uint
	FileLength  int64
	HashLengths ControlHeaderHashLengths
	URL         string
	SHA1        string

	ChecksumIndex *index.ChecksumIndex
}

func ReadControl(input io.Reader) (control *Control, err error) {
	control = &Control{}
	reader := bufio.NewReader(input)
	err = control.readHeaders(reader)
	if err != nil {
		return nil, err
	}

	err = control.readChecksums(reader)

	return
}

func (control *Control) readHeaders(reader *bufio.Reader) error {
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return err
		}

		// the header end is marked by an empty line "\n"
		if line == "\n" {
			break
		}

		k, v := parseHeaderLine(line)
		setHeaderValue(control, k, v)
	}
	return nil
}

func setHeaderValue(c *Control, k string, v string) {
	switch k {
	case "zsync":
		c.Version = v
	case "filename":
		c.FileName = v
	case "mtime":
		c.MTime = v
	case "blocksize":
		vi, err := strconv.ParseUint(v, 10, 0)
		if err == nil {
			c.BlockSize = uint(vi)
		}

	case "length":
		vi, err := strconv.ParseInt(v, 10, 0)
		if err == nil {
			c.FileLength = vi
		}
	case "hash-lengths":
		hashLenghts, err := parseHashLengths(v)
		if err == nil {
			c.HashLengths = *hashLenghts
		}
	case "url":
		c.URL = v
	case "sha-1":
		c.SHA1 = v
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

func (control *Control) readChecksums(reader io.Reader) error {
	readChunks, err := chunks.LoadChecksumsFromReaderLegacy(
		reader,
		int(control.HashLengths.WeakCheckSumBytes),
		int(control.HashLengths.StrongCheckSumBytes),
	)

	if err != nil {
		return err
	}

	control.ChecksumIndex = index.MakeChecksumIndex(readChunks,
		control.HashLengths.WeakCheckSumBytes,
		control.HashLengths.StrongCheckSumBytes)

	control.Blocks = uint(control.ChecksumIndex.BlockCount)

	return nil
}
