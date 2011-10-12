
package blocks

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"strings"
	"container/vector"
)

const BLOCKSIZE int = 8192

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

// Nodes are any member of a hierarchical index.
type Node interface {
	
	// Test if this node is at the root of the index.
	IsRoot() bool
	
	// Get the strong checksum of a node.
	Strong() string
	
	// Get the node that contains this node in the hierarchical index.
	Parent() FsNode
	
}

// FsNodes are members of a hierarchical index that correlate to the filesystem.
type FsNode interface {
	
	// FsNode extends the concept of Node.
	Node
	
	// FsNodes all have names.
	Name() string
	
}

func RelPath(item FsNode) string {
	parts := vector.StringVector{}
	
	for fsNode := item; !fsNode.IsRoot(); fsNode = fsNode.Parent() {
		parts.Insert(0, fsNode.Name())
	}
	
	return filepath.Join(parts...)
}

// Represent a block in a hierarchical index.
type Block struct {
	position int
	weak int
	strong string
	parent *File
}

// Get the weak checksum of a block.
func (block *Block) Weak() int { return block.weak }

// Get the position of the block in its containing file
func (block *Block) Position() int { return block.position }

func (block *Block) Offset() int64 { return int64(block.position) * int64(BLOCKSIZE) }

func (block *Block) IsRoot() (bool) { return false }

func (block *Block) Strong() (string) { return block.strong }

func (block *Block) Parent() (FsNode) { return block.parent }

func (block *Block) Child(i int) (Node) { return nil }

func (block *Block) ChildCount() (int) { return 0 }

// Represent a file in a hierarchical index.
type File struct {
	name string
	strong string
	parent *Dir
	
	Size int64
	Blocks []*Block
}

func (file *File) Name() (string) { return file.name }

func (file *File) IsRoot() (bool) { return false }

func (file *File) Strong() (string) { return file.strong }

func (file *File) Parent() (FsNode) { return file.parent }

// Represent a directory in a hierarchical index.
type Dir struct {
	name string
	strong string
	parent *Dir
	
	SubDirs []*Dir
	Files []*File
}

func (dir *Dir) Name() (string) { return dir.name }

func (dir *Dir) IsRoot() (bool) { return dir.parent == nil }

func (dir *Dir) Strong() (string) {
	if dir.strong == "" {
		dir.strong = dir.calcStrong()
	}
	return dir.strong
}

func (dir *Dir) calcStrong() string {
	var sha1 = sha1.New()
	sha1.Write(dir.stringBytes())
	return toHexString(sha1)
}

func (dir *Dir) Parent() (FsNode) { return dir.parent }

func (dir *Dir) stringBytes() []byte {
	buf := bytes.NewBufferString("")
	
	for _, subdir := range dir.SubDirs {
		fmt.Fprintf(buf, "%s\td\t%s\n", subdir.Strong(), subdir.Name())
	}
	for _, file := range dir.Files {
		fmt.Fprintf(buf, "%s\tf\t%s\n", file.Strong(), file.Name())
	}
	
	return buf.Bytes()
}

// Represent the directory as a string describing its entries, with strong checksums.
func (dir *Dir) String() string	{
	return string(dir.stringBytes())
}

// Visitor function that is used to traverse a hierarchical Node index.
type NodeVisitor func(Node) bool

// Traverse a hierarchical Node index with user-defined NodeVisitor function.
func Walk(node Node, visitor NodeVisitor) {
	nodestack := []Node{}
	nodestack = append(nodestack, node)
	
	for ; len(nodestack) > 0 ; {
		current := nodestack[0]
		nodestack = nodestack[1:]
		if visitor(current) {
			
			if dir, isDir := current.(*Dir); isDir {
				for _, subdir := range dir.SubDirs {
					nodestack = append(nodestack, subdir)
				}
				for _, file := range dir.Files {
					nodestack = append(nodestack, file)
				}
			} else if file, isFile := current.(*File); isFile {
				for _, block := range file.Blocks {
					nodestack = append(nodestack, block)
				}
			}
			
		}
	}
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

// Provide access to the raw byte storage.
type BlockStore interface {
	
	// Get the root hierarchical index node
	Root() *Dir
	
	Index() *BlockIndex
	
	// Given a strong checksum of a block, get the bytes for that block.
	ReadBlock(strong string) ([]byte, os.Error)
	
	// Given the strong checksum of a file, start and end positions, get those bytes.
	ReadInto(strong string, from int64, length int64, writer io.Writer) os.Error
	
}

// A local file implementation of BlockStore
type LocalStore struct {
	rootPath string
	root *Dir
	index *BlockIndex
}

func NewLocalStore(rootPath string) (*LocalStore, os.Error) {
	local := &LocalStore{rootPath:rootPath}
	
	var err os.Error
	
	local.root, err = IndexDir(rootPath)
	if local.root == nil { return nil, err }
	
	local.index = IndexBlocks(local.root)
	return local, nil
}

func (store *LocalStore) LocalPath(relpath string) string {
	return filepath.Join(store.rootPath, relpath)
}

func (store *LocalStore) Root() *Dir { return store.root }

func (store *LocalStore) Index() *BlockIndex { return store.index }

func (store *LocalStore) ReadBlock(strong string) ([]byte, os.Error) {
	maybeBlock, has := store.index.StrongMap[strong]
	if !has { 
		return nil, os.NewError(
				fmt.Sprintf("Block with strong checksum %s not found", strong))
	}
	
	block, is := maybeBlock.(*Block)
	if !is { return nil, os.NewError(fmt.Sprintf("%s: not a block", strong)) }
	
	buf := &bytes.Buffer{}
	err := store.ReadInto(block.Parent().Strong(), block.Offset(), int64(BLOCKSIZE), buf)
	if err == nil {
		return nil, err
	}
	
	return buf.Bytes(), nil
}

func (store *LocalStore) ReadInto(strong string, from int64, length int64, writer io.Writer) os.Error {
	
	node, has := store.index.StrongMap[strong]
	if !has {
		return os.NewError(fmt.Sprintf("File with strong checksum %s not found", strong))
	}
	
	file, is := node.(*File)
	if !is { return os.NewError(fmt.Sprintf("%s: not a file", strong)) }
	
	path := store.LocalPath(RelPath(file))
	
	fh, err := os.Open(path)
	if fh == nil { return err }
	
	_, err = fh.Seek(from, 0)
	if err != nil { return err }
	
	_, err = io.Copyn(writer, fh, length)
	if err != nil {
		return err
	}
	
	return nil
}



