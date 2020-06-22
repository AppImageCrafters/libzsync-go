package zsync

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"sort"

	"golang.org/x/crypto/md4"

	"github.com/AppImageCrafters/zsync/chunks"
	"github.com/AppImageCrafters/zsync/control"
	"github.com/AppImageCrafters/zsync/rollsum"
	"github.com/AppImageCrafters/zsync/sources"
)

type SyncData struct {
	control.Control

	WeakChecksumBuilder   hash.Hash
	StrongChecksumBuilder hash.Hash
	Local                 *os.File
	Output                io.Writer
	progress              ProgressReporter
}

func Sync(local *os.File, output io.Writer, control control.Control, progressReporter ProgressReporter) (err error) {
	syncData := SyncData{
		Control:               control,
		WeakChecksumBuilder:   rollsum.NewRollsum32(control.BlockSize),
		StrongChecksumBuilder: md4.New(),
		Local:                 local,
		Output:                output,
		progress:              progressReporter,
	}

	reusableChunks, err := syncData.searchReusableChunks()
	if err != nil {
		return
	}

	syncData.printChunksSummary(reusableChunks)
	allChunks := syncData.AddMissingChunks(reusableChunks)

	err = syncData.mergeChunks(allChunks, output)
	return
}

func (syncData *SyncData) mergeChunks(allChunks []chunks.ChunkInfo, output io.Writer) error {
	outputSHA1 := sha1.New()

	syncData.progress.SetDescription("Merging chunks: ")
	syncData.progress.SetTotal(syncData.FileLength)

	for _, chunk := range allChunks {
		_, err := chunk.Source.Seek(chunk.SourceOffset, 0)
		if err != nil {
			return err
		}

		// request the whole chunks in advance avoid small request
		httpFileSource, ok := chunk.Source.(*sources.HttpFileSource)
		if ok {
			err = httpFileSource.Request(chunk.Size)
		}

		_, err = io.CopyN(io.MultiWriter(output, outputSHA1, syncData.progress), chunk.Source, chunk.Size)
		if err != nil {
			return err
		}
	}

	outputSHA1Sum := hex.EncodeToString(outputSHA1.Sum(nil))
	if outputSHA1Sum != syncData.SHA1 {
		return fmt.Errorf("Output checksum don't match with the expected")
	}
	return nil
}

func (syncData *SyncData) searchReusableChunks() (matchingChunks []chunks.ChunkInfo, err error) {
	matchingChunks, err = syncData.SearchAllMatchingChunks()
	if err != nil {
		return nil, err
	}

	matchingChunks = removeDuplicatedChunks(matchingChunks)
	matchingChunks = removeSmallChunks(matchingChunks, syncData.FileLength)

	return
}

func (syncData *SyncData) printChunksSummary(matchingChunks []chunks.ChunkInfo) {
	reusableChunksSize := int64(0)
	for _, chunk := range matchingChunks {
		reusableChunksSize += chunk.Size
	}
	fmt.Printf("Reusable chunks found: %d %dKb (%d%%)\n",
		len(matchingChunks), reusableChunksSize/1024, reusableChunksSize*100/syncData.FileLength)
}

func removeSmallChunks(matchingChunks []chunks.ChunkInfo, length int64) (filteredChunks []chunks.ChunkInfo) {
	for _, chunk := range matchingChunks {
		if chunk.Size > 1024 || chunk.TargetOffset+chunk.Size == length {
			filteredChunks = append(filteredChunks, chunk)
		}
	}

	return
}

func removeDuplicatedChunks(matchingChunks []chunks.ChunkInfo) []chunks.ChunkInfo {
	m := make(map[int64]chunks.ChunkInfo)
	for _, item := range matchingChunks {
		if _, ok := m[item.TargetOffset]; ok {
			// prefer chunks with the same offset in both files
			if item.SourceOffset == item.TargetOffset {
				m[item.TargetOffset] = item
			}
		} else {
			m[item.TargetOffset] = item
		}
	}

	var result []chunks.ChunkInfo
	for _, item := range m {
		result = append(result, item)
	}

	return result
}

func sortChunksByTargetOffset(matchingChunks []chunks.ChunkInfo) {
	sort.Slice(matchingChunks, func(i, j int) bool {
		return matchingChunks[i].TargetOffset < matchingChunks[j].TargetOffset
	})
}

func (syncData *SyncData) AddMissingChunks(matchingChunks []chunks.ChunkInfo) (missing []chunks.ChunkInfo) {
	sortChunksByTargetOffset(matchingChunks)
	missingChunksSource := sources.HttpFileSource{URL: syncData.URL, Size: syncData.FileLength}

	offset := int64(0)
	for _, chunk := range matchingChunks {
		gapSize := chunk.TargetOffset - offset
		if gapSize > 0 {
			if chunk.TargetOffset != offset {
				missingChunk := chunks.ChunkInfo{
					Size:         gapSize,
					Source:       &missingChunksSource,
					SourceOffset: offset,
					TargetOffset: offset,
				}

				missing = append(missing, missingChunk)
				offset += gapSize
			}
		}

		missing = append(missing, chunk)
		offset += chunk.Size
	}

	if offset < syncData.FileLength {
		missingChunk := chunks.ChunkInfo{
			Size:         syncData.FileLength - offset,
			Source:       &missingChunksSource,
			SourceOffset: offset,
			TargetOffset: offset,
		}

		missing = append(missing, missingChunk)
	}

	return
}
