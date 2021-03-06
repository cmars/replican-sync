package sync

import (
	"fmt"
	"os"
	"github.com/cmars/replican-sync/replican/fs"
)

type BlockMatch struct {
	SrcBlock  fs.Block
	DstOffset int64
}

type FileMatch struct {
	SrcSize      int64
	DstSize      int64
	BlockMatches []*BlockMatch
}

type RangePair struct {
	From int64
	To   int64
}

func (r *RangePair) Size() int64 {
	return r.To - r.From
}

func Match(src string, dst string) (match *FileMatch, err os.Error) {
	srcFileInfo, srcBlocksInfo, err := fs.IndexFile(src)
	if err != nil {
		return nil, err
	}

	repo := fs.NewMemRepo()
	srcFile := repo.AddFile(nil, srcFileInfo, srcBlocksInfo)
	match, err = MatchFile(srcFile, dst)
	return match, err
}

func MatchFile(srcFile fs.File, dst string) (match *FileMatch, err os.Error) {
	match = &FileMatch{SrcSize: srcFile.Info().Size}
	var dstOffset int64

	dstF, err := os.Open(dst)
	if dstF == nil {
		return nil, err
	}
	defer dstF.Close()

	if dstInfo, err := dstF.Stat(); dstInfo == nil {
		return nil, err
	} else if !dstInfo.IsRegular() {
		return nil, os.NewError(fmt.Sprintf("%s: not a regular file", dst))
	} else {
		match.DstSize = dstInfo.Size
	}

	dstWeak := new(fs.WeakChecksum)
	var buf [fs.BLOCKSIZE]byte
	var sbuf [1]byte
	var window []byte

	// Scan a block,
	// then roll checksum a byte at a time until match or eof
	// repeat above until eof
SCAN:
	for {
		switch rd, err := dstF.Read(buf[:]); true {
		case rd < 0:
			return nil, err

		case rd == 0:
			break SCAN

		case rd > 0:
			blocksize := rd
			dstOffset += int64(rd)
			window = buf[:rd]

			dstWeak.Reset()
			dstWeak.Write(window[:])

			for {
				// Check for a weak checksum match
				if matchBlock, has := srcFile.Repo().WeakBlock(dstWeak.Get()); has {

					// Double-check with the strong checksum
					if fs.StrongChecksum(window[:blocksize]) == matchBlock.Info().Strong {

						// We've got a block match in dest
						match.BlockMatches = append(match.BlockMatches, &BlockMatch{
							SrcBlock:  matchBlock,
							DstOffset: dstOffset - int64(blocksize)})
						break
					}
				}

				// Read the next byte
				switch srd, err := dstF.Read(sbuf[:]); true {
				case srd < 0:
					return nil, err

				case srd == 0:
					break SCAN

				case srd == 1:
					dstOffset++

					// Roll the weak checksum & the buffer
					dstWeak.Roll(window[0], sbuf[0])
					window = append(window[1:], sbuf[0])
					break

				case srd > 1:
					return nil, os.NewError("Internal read error trying advance one byte.")
				}
			}
		}
	}

	return match, nil
}

func (match *FileMatch) NotMatched() (ranges []*RangePair) {
	start := int64(0)

	for _, blockMatch := range match.BlockMatches {
		if start < blockMatch.DstOffset {
			ranges = append(ranges, &RangePair{From: start, To: blockMatch.DstOffset})
		}
		start = blockMatch.DstOffset + int64(fs.BLOCKSIZE)
	}

	if start < match.SrcSize {
		ranges = append(ranges, &RangePair{From: start, To: match.SrcSize})
	}

	return ranges
}
