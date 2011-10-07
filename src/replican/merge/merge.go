
package merge

import (
	"os"
	"replican/blocks"
)

/*
type MoveFile struct {}
type DeleteFile struct {}
type TruncateFile struct {}
type AddBlock struct {}
type CopyBlock struct {}
*/

type BlockMatch struct {
	SrcBlock *blocks.Block
	DstOffset int64
}

type FileMatch struct {
	SrcSize int64
	DstSize int64
	BlockMatches []*BlockMatch
}

func Match(src string, dst string) (match *FileMatch, err os.Error) {
	var srcFile *blocks.File
	srcFile, err = blocks.IndexFile(src)
	if srcFile == nil {
		return nil, err
	}
	
	match, err = MatchFile(srcFile, dst)
	return match, err
}

func MatchFile(srcFile *blocks.File, dst string) (match *FileMatch, err os.Error) {
	srcBlockIndex := blocks.IndexBlocks(srcFile)
	
	match, err = MatchIndex(srcBlockIndex, dst)
	if match != nil {
		match.SrcSize = srcFile.Size()
	}
	
	return match, err
}

func MatchIndex(srcBlockIndex *blocks.BlockIndex, dst string) (match *FileMatch, err os.Error) {
	match = new(FileMatch)
	var dstOffset int64
	
	var dstF *os.File
	dstF, err = os.Open(dst)
	if dstF == nil {
		return nil, err
	}
	
	if dstInfo, err := dstF.Stat(); dstInfo != nil {
		match.DstSize = dstInfo.Size
	} else {
		return nil, err
	}
	
	dstWeak := new(blocks.WeakChecksum)
	var buf [blocks.BLOCKSIZE]byte
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
				if matchBlock, has := srcBlockIndex.WeakMap[dstWeak.Get()]; has {
					
					// Double-check with the strong checksum
					if blocks.StrongChecksum(window[:blocksize]) == matchBlock.Strong() {
						// map this match
						match.BlockMatches = append(match.BlockMatches, &BlockMatch{
							SrcBlock:matchBlock, 
							DstOffset: dstOffset - int64(blocksize) })
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



