
package blocks

import (
	"crypto/sha1"
	"fmt"
	"hash"
	"os"
	"path/filepath"
	"strings"
)

// Store a weak checksum as described in the rsync algorithm paper
type WeakChecksum struct {
	a int
	b int
}

// Reset the state of the checksum
func (weak *WeakChecksum) Reset() {
	weak.a = 0
	weak.b = 0
}

// Write a block of data into the checksum
func (weak *WeakChecksum) Write(buf []byte) {
	for i := 0; i < len(buf); i++ {
		b := int(buf[i])
		weak.a += b;
		weak.b += (len(buf) - i) * b;
	}
}

// Get the current weak checksum value
func (weak *WeakChecksum) Get() int {
	return weak.b << 16 | weak.a;
}

// Roll the checksum forward by one byte
func (weak *WeakChecksum) Roll(removedByte byte, newByte byte) {
    weak.a -= int(removedByte) - int(newByte);
    weak.b -= int(removedByte) * BLOCKSIZE - weak.a;
}

// Visitor used to traverse a directory with filepath.Walk in IndexDir
type indexVisitor struct {
	root *Dir
	currentDir *Dir
	dirMap map[string]*Dir
}

// Initialize the IndexDir visitor
func newVisitor(path string) *indexVisitor {
	path = filepath.Clean(path)
	path = strings.TrimRight(path, "/\\")
	
	visitor := new(indexVisitor)
	visitor.dirMap = make(map[string]*Dir)
	visitor.root = new(Dir)
	visitor.currentDir = visitor.root
	visitor.dirMap[path] = visitor.root
	
	return visitor
}

// IndexDir visitor callback for directories
func (visitor *indexVisitor) VisitDir(path string, f *os.FileInfo) bool {
	path = filepath.Clean(path)
	
	dir, hasDir := visitor.dirMap[path]
	if !hasDir {
		dir = new(Dir)
		visitor.dirMap[path] = dir
		
		dirname, basename := filepath.Split(path)
		dirname = strings.TrimRight(dirname, "/\\") // remove the trailing slash
		
		dir.name = basename
		dir.parent = visitor.dirMap[dirname]
		
		if dir.parent != nil {
			dir.parent.SubDirs = append(dir.parent.SubDirs, dir)
		}
	}
		
	visitor.currentDir = dir;
	return true
}

// IndexDir visitor callback for files
func (visitor *indexVisitor) VisitFile(path string, f *os.FileInfo) {
	file, err := IndexFile(path)
	if file != nil {
		file.parent = visitor.currentDir
		visitor.currentDir.Files = append(visitor.currentDir.Files, file)
	} else {
		fmt.Errorf("failed to read file %s: %s", path, err.String())
	}
}

// Build a hierarchical index of a directory
func IndexDir(path string) (dir *Dir, err os.Error) {
	visitor := newVisitor(path)
	filepath.Walk(path, visitor, nil)
	if visitor.root != nil {
		visitor.root.Strong()
		return visitor.root, nil
	}
	return nil, nil
}

// Build a hierarchical index of a file
func IndexFile(path string) (file *File, err os.Error) {
	var f *os.File
	var buf [BLOCKSIZE]byte
	
	f, err = os.Open(path)
	if f == nil {
		return nil, err
	}
	defer f.Close()
	
	file = new(File)
	_, basename := filepath.Split(path)
	file.name = basename
	
	if fileInfo, err := f.Stat(); fileInfo != nil {
		file.Size = fileInfo.Size
	} else {
		return nil, err
	}
	
	var block *Block
	sha1 := sha1.New()
	blockNum := 0
	
	for {
		switch rd, err := f.Read(buf[:]); true {
		case rd < 0:
			return nil, err
		case rd == 0:
			file.strong = toHexString(sha1)
			return file, nil
		case rd > 0:
			// Update block hashes
			block = IndexBlock(buf[0:rd])
			block.position = blockNum
			file.Blocks = append(file.Blocks, block)
			
			// update file hash
			sha1.Write(buf[0:rd])
			
			// Increment block counter
			blockNum++
		}
	}
	
	return nil, nil
}

// Render a Hash as a hexadecimal string
func toHexString(hash hash.Hash) string {
	return fmt.Sprintf("%x", hash.Sum())
}

// Strong checksum algorithm used throughout replican
// For now, it's SHA-1
func StrongChecksum(buf []byte) string {
	var sha1 = sha1.New()
	sha1.Write(buf)
	return toHexString(sha1)
}

// Index a block of data with weak and strong checksums
func IndexBlock(buf []byte) (block *Block) {
	block = new(Block)
	
	var weak = new(WeakChecksum)
	weak.Write(buf)
	block.weak = weak.Get()
	
	block.strong = StrongChecksum(buf)
	
	return block
}

// Represent a flat mapping between checksum and Nodes in a hierarchical index.
type BlockIndex struct {
	WeakMap map[int]*Block 
	StrongMap map[string]Node
}

// Derive a flattened BlockIndex from a top-level Node.
func IndexBlocks(node Node) (index *BlockIndex) {
	index = new(BlockIndex)
	index.WeakMap = make(map[int]*Block)
	index.StrongMap = make(map[string]Node)
	
	Walk(node, func(current Node) bool {
		index.StrongMap[current.Strong()] = current
		if block, isblock := current.(*Block); isblock {
			index.WeakMap[block.Weak()] = block
		}
		return true
	})
	
	return index
}


