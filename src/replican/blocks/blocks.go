
package blocks

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"path/filepath"
	"container/vector"
)

const BLOCKSIZE int = 8192

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



