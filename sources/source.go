package sources

import (
	"io"
)

func ReadChunk(source io.ReadSeeker, offset int64, requiredBytes int64) (blockData []byte, err error) {
	_, err = source.Seek(offset, 0)
	if err != nil {
		return nil, err
	}

	reader := io.LimitedReader{source, requiredBytes}
	blockData = make([]byte, requiredBytes)
	n, err := reader.Read(blockData)
	if int64(n) == requiredBytes {
		return blockData, nil
	}

	if err != nil {
		return nil, err
	}

	return
}
