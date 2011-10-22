
package fs

import (
	"crypto/sha1"
	"fmt"
	"hash"
	"os"
	"path/filepath"
	"strings"
)

// Represent a weak checksum as described in the rsync algorithm paper
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

// Build a hierarchical tree model representing a directory's contents
func IndexDir(path string) (dir *Dir, err os.Error) {
	visitor := newVisitor(path)
	filepath.Walk(path, visitor, nil)
	if visitor.root != nil {
		visitor.root.Strong()
		return visitor.root, nil
	}
	return nil, nil
}

// Build a hierarchical tree model representing a file's contents
func IndexFile(path string) (file *File, err os.Error) {
	var f *os.File
	var buf [BLOCKSIZE]byte
	
	if stat, err := os.Stat(path); stat == nil {
		return nil, err
	} else if !stat.IsRegular() {
		return nil, os.NewError(fmt.Sprintf("%s: not a regular file", path))
	}
	
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

// Render a Hash as a hexadecimal string.
func toHexString(hash hash.Hash) string {
	return fmt.Sprintf("%x", hash.Sum())
}

// Strong checksum algorithm used throughout replican
// For now, it's SHA-1.
func StrongChecksum(buf []byte) string {
	var sha1 = sha1.New()
	sha1.Write(buf)
	return toHexString(sha1)
}

// Model a block with weak and strong checksums.
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
	weakBlocks map[int]*Block 
	strongBlocks map[string]*Block
	strongFiles map[string]*File
	strongDirs map[string]*Dir
}

// Get the Block with matching weak checksum.
// Boolean return value indicates if a match was found.
func (index *BlockIndex) WeakBlock(weak int) (block *Block, has bool) {
	block, has = index.weakBlocks[weak]
	return block, has
}

// Get the filesystem node with matching strong checksum.
// Boolean return value indicates if a match was found.
func (index *BlockIndex) StrongFsNode(strong string) (FsNode, bool) {
	file, has := index.strongFiles[strong]
	if has { return file, true }
	
	dir, has := index.strongDirs[strong]
	if has { return dir, true }
	
	return nil, false
}

// Get the block with matching strong checksum.
// Boolean return value indicates if a match was found.
func (index *BlockIndex) StrongBlock(strong string) (block *Block, has bool) {
	block, has = index.strongBlocks[strong]
	return block, has
}

// Get the file with matching strong checksum.
// Boolean return value indicates if a match was found.
func (index *BlockIndex) StrongFile(strong string) (file *File, has bool) {
	file, has = index.strongFiles[strong]
	return file, has
}

// Get the directory with matching strong checksum.
// Boolean return value indicates if a match was found.
func (index *BlockIndex) StrongDir(strong string) (dir *Dir, has bool) {
	dir, has = index.strongDirs[strong]
	return dir, has
}

// Derive a flattened BlockIndex from a top-level Node.
// This index maps checksums to the corresponding hierarchical model.
func IndexBlocks(node Node) (index *BlockIndex) {
	index = new(BlockIndex)
	index.weakBlocks = make(map[int]*Block)
	index.strongBlocks = make(map[string]*Block)
	index.strongFiles = make(map[string]*File)
	index.strongDirs = make(map[string]*Dir)
	
	Walk(node, func(current Node) bool {
		switch t := current.(type) {
		case *Block:
			block := current.(*Block)
			index.strongBlocks[current.Strong()] = block
			index.weakBlocks[block.Weak()] = block
			return false
		case *File:
			index.strongFiles[current.Strong()] = current.(*File)
		case *Dir:
			index.strongDirs[current.Strong()] = current.(*Dir)
		}
		return true
	})
	
	return index
}



