package replican

import (
	"crypto/sha1"
/*
	"bytes"
	"fmt"
	"path/filepath"
*/
)

// Block size used for checksum, comparison, transmitting deltas.
const BLOCKSIZE int = 8192

const (
	_ = iota
	BLOCK
	FILE
	DIR
)

type RecType byte

// Represent a block in a hierarchical tree model.
// Blocks are BLOCKSIZE chunks of data which comprise files.
type BlockRec struct {
	Type     RecType
	Strong   [sha1.Size]byte
	Weak     int
	Position int
}

// Get the byte offset of this block in its containing file.
func (block *BlockRec) Offset() int64 {
	return int64(block.Position) * int64(BLOCKSIZE)
}

// Represent a file in a hierarchical tree model.
type FileRec struct {
	Type    RecType
	Strong  [sha1.Size]byte
	Sibling int
	Depth   int
}

// Represent a directory in a hierarchical tree model.
type DirRec struct {
	Type    RecType
	Strong  [sha1.Size]byte
	Sibling int
	Depth   int
}