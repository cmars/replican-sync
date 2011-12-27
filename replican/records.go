package replican

import (
	"crypto/sha1"
)

// Block size used for checksum, comparison, transmitting deltas.
const BLOCKSIZE int = 8192

const RECSIZE int = 1+sha1.Size+4+4

const (
	_ = iota
	BLOCK
	FILE
	DIR
)

type RecType uint8

func (recType RecType) String() string {
	switch (recType) {
	case BLOCK:
		return "BLOCK"
	case FILE:
		return "FILE"
	case DIR:
		return "DIR"
	}
	return "UNKNOWN"
}

// Represent a block in a hierarchical tree model.
// Blocks are BLOCKSIZE chunks of data which comprise files.
type BlockRec struct {
	Type     uint8
	Strong   [sha1.Size]byte
	Weak     int32
	Position int32
}

// Get the byte offset of this block in its containing file.
func (block *BlockRec) Offset() int64 {
	return int64(block.Position) * int64(BLOCKSIZE)
}

// Represent a file in a hierarchical tree model.
type FileRec struct {
	Type    uint8
	Strong  [sha1.Size]byte
	Sibling int32
	Depth   int32
}

// Represent a directory in a hierarchical tree model.
type DirRec struct {
	Type    uint8
	Strong  [sha1.Size]byte
	Sibling int32
	Depth   int32
}

// Message containing a scanned record during a directory walk.
// Only one of Block, File or Dir will be set, the others will be nil.
type ScanRec struct {

	// The sequence number is the order in which the record occurs.
	// This alone can be used to locate the record in the stream since records 
	// have a fixed size.
	Seq int32
	
	// The path associated with the record.
	// If the record is a block, the path is an empty string.
	Path string
	
	Block *BlockRec
	File *FileRec
	Dir *DirRec
}

type RecSource interface {
	
	Records() <- chan *ScanRec
	
}
