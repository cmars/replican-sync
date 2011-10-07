
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
	DstOffset uint
}

func MatchFiles(src string, dst string) (matches []*BlockMatch, err os.Error) {
	dstOffset := 0
	
	var srcFile *blocks.File
	srcFile, err = blocks.IndexFile(src)
	if srcFile == nil {
		return nil, err
	}
	
	srcBlockIndex := blocks.IndexBlocks(srcFile)
	
	var dstF *os.File
	dstF, err = os.Open(dst)
	if dstF == nil {
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
			dstOffset += rd
			window = buf[:rd]
			
			dstWeak.Reset()
			dstWeak.Write(buf[:rd])
			
			for {
				// Check for a weak checksum match
				if matchBlock, has := srcBlockIndex.WeakMap[dstWeak.Get()]; has {
					
					// Double-check with the strong checksum
					if blocks.StrongChecksum(buf[:rd]) == matchBlock.Strong() {
						// map this match
						matches = append(matches, &BlockMatch{
							SrcBlock:matchBlock, 
							DstOffset: uint(dstOffset - blocksize) })
						break
					}
				}
				
				// Read the next byte
				switch srd, err := dstF.Read(sbuf[:]); true {
				case srd < 0:
					return nil, err
				case srd == 0:
					break SCAN
				
				case srd > 0:
					dstOffset += srd
					
					// Roll the weak checksum & the buffer
					dstWeak.Roll(buf[0], sbuf[0])
					window = append(window[1:], sbuf[0])
				}
			}
		}
	}
	
	return matches, nil
}



