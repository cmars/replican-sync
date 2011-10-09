
package blocks

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"hash"
	"os"
	"path/filepath"
	"strings"
)

const BLOCKSIZE int = 8192

type WeakChecksum struct {
	a int
	b int
}

func (weak *WeakChecksum) Reset() {
	weak.a = 0
	weak.b = 0
}

func (weak *WeakChecksum) Write(buf []byte) {
	for i := 0; i < len(buf); i++ {
		b := int(buf[i])
		weak.a += b;
		weak.b += (len(buf) - i) * b;
	}
}

func (weak *WeakChecksum) Get() int {
	return weak.b << 16 | weak.a;
}

func (weak *WeakChecksum) Roll(removedByte byte, newByte byte) {
    weak.a -= int(removedByte) - int(newByte);
    weak.b -= int(removedByte) * BLOCKSIZE - weak.a;
}

type indexVisitor struct {
	root *Dir
	currentDir *Dir
	dirMap map[string]*Dir
}

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
			dir.parent.subdirs = append(dir.parent.subdirs, dir)
		}
	}
		
	visitor.currentDir = dir;
	return true
}

func (visitor *indexVisitor) VisitFile(path string, f *os.FileInfo) {
	file, err := IndexFile(path)
	if file != nil {
		file.parent = visitor.currentDir
		visitor.currentDir.files = append(visitor.currentDir.files, file)
	} else {
		fmt.Errorf("failed to read file %s: %s", path, err.String())
	}
}

func IndexDir(path string) (dir *Dir, err os.Error) {
	visitor := newVisitor(path)
	filepath.Walk(path, visitor, nil)
	if visitor.root != nil {
		visitor.root.Strong()
		return visitor.root, nil
	}
	return nil, nil
}

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
		file.size = fileInfo.Size
	} else {
		return nil, err
	}
	
	var block *Block
	var sha1 = sha1.New()
	
	for {
		switch rd, err := f.Read(buf[:]); true {
		case rd < 0:
			return nil, err
		case rd == 0:
			file.strong = toHexString(sha1)
			return file, nil
		case rd > 0:
			// update block hashes
			block = IndexBlock(buf[0:rd])
			file.blocks = append(file.blocks, block)
			
			// update file hash
			sha1.Write(buf[0:rd])
		}
	}
	
	return nil, nil
}

func toHexString(hash hash.Hash) string {
	return fmt.Sprintf("%x", hash.Sum())
}

func StrongChecksum(buf []byte) string {
	var sha1 = sha1.New()
	sha1.Write(buf)
	return toHexString(sha1)
}

func IndexBlock(buf []byte) (block *Block) {
	block = new(Block)
	
	var weak = new(WeakChecksum)
	weak.Write(buf)
	block.weak = weak.Get()
	
	block.strong = StrongChecksum(buf)
	
	return block
}

type Node interface {
	IsRoot() bool
	Strong() string
	Parent() Node
	Child(i int) Node
	ChildCount() int
}

type FsNode interface {
	Node
	Name() string
}

type Block struct {
	weak int
	strong string
	parent *File
}

func (block *Block) Weak() int { return block.weak }

func (block *Block) IsRoot() (bool) { return false }

func (block *Block) Strong() (string) { return block.strong }

func (block *Block) Parent() (Node) { return block.parent }

func (block *Block) Child(i int) (Node) { return nil }

func (block *Block) ChildCount() (int) { return 0 }

type File struct {
	name string
	size int64
	strong string
	parent *Dir
	blocks []*Block
}

func (file *File) Name() (string) { return file.name }

func (file *File) IsRoot() (bool) { return false }

func (file *File) Size() int64 { return file.size }

func (file *File) Strong() (string) { return file.strong }

func (file *File) Parent() (Node) { return file.parent }

func (file *File) Child(i int) (Node) { return file.blocks[i] }

func (file *File) ChildCount() (int) { return len(file.blocks) }

type Dir struct {
	name string
	strong string
	parent *Dir
	subdirs []*Dir
	files []*File
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

func (dir *Dir) Parent() (Node) { return dir.parent }

func (dir *Dir) Child(i int) (Node) {
	switch sl := len(dir.subdirs); true {
	case i < sl:
		return dir.subdirs[i]
	default:
		return dir.files[i-sl]
	}
	return nil
}

func (dir *Dir) stringBytes() []byte {
	buf := bytes.NewBufferString("")
	
	for _, subdir := range dir.subdirs {
		fmt.Fprintf(buf, "%s\td\t%s\n", subdir.Strong(), subdir.Name())
	}
	for _, file := range dir.files {
		fmt.Fprintf(buf, "%s\tf\t%s\n", file.Strong(), file.Name())
	}
	
	return buf.Bytes()
}

func (dir *Dir) String() string	{
	return string(dir.stringBytes())
}

func (dir *Dir) ChildCount() (int) { return len(dir.subdirs) + len(dir.files) }

type NodeVisitor func(Node) bool

func Walk(node Node, visitor NodeVisitor) {
	nodestack := []Node{}
	nodestack = append(nodestack, node)
	
	for ; len(nodestack) > 0 ; {
		current := nodestack[0]
		nodestack = nodestack[1:]
		if visitor(current) {
			for i := 0; i < current.ChildCount(); i++ {
				nodestack = append(nodestack, current.Child(i))
			}
		}
	}
}

type BlockIndex struct {
	WeakMap map[int]*Block 
	StrongMap map[string]Node
}

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



