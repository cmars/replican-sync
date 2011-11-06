package fs

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"path/filepath"
)

// Block size used for checksum, comparison, transmitting deltas.
const BLOCKSIZE int = 8192

// Nodes are any member of a hierarchical tree model representing 
// a part of the filesystem. Nodes include files and directories,
// and also blocks within the files.
type Node interface {

	// Get the strong checksum of a node.
	Strong() string

	// Get the node that contains this node in the hierarchical index.
	Parent() Node
}

// FsNodes are members of a hierarchical index that map directly onto the filesystem:
// files and directories.
type FsNode interface {

	// Test if this node is at the root of the tree.
	IsRoot() bool

	// FsNode extends the concept of Node.
	Node

	// All FsNodes have names (file or directory name).
	Name() string

	// All FsNodes have permissions
	Mode() uint32

	fsParent() FsNode
}

// Given a filesystem node, calculate the relative path string to it from the root node.
func RelPath(item FsNode) string {
	parts := []string{}

	for fsNode := item; !fsNode.IsRoot(); fsNode = fsNode.fsParent() {
		parts = append([]string{fsNode.Name()}, parts...)
	}

	return filepath.Join(parts...)
}

// Represent a block in a hierarchical tree model.
// Blocks are BLOCKSIZE chunks of data which comprise files.
type Block struct {
	position int
	weak     int
	strong   string
	parent   *File
}

// Get the weak checksum of a block.
func (block *Block) Weak() int { return block.weak }

// Get the ordinal position of the block in its containing file.
// For example, the block beginning at byte offset BLOCKSIZE*2 would be position 2.
func (block *Block) Position() int { return block.position }

// Get the byte offset of this block in its containing file.
func (block *Block) Offset() int64 { return int64(block.position) * int64(BLOCKSIZE) }

// Blocks are never root nodes.
func (block *Block) IsRoot() bool { return false }

func (block *Block) Strong() string { return block.strong }

func (block *Block) Parent() Node { return block.parent }

func (block *Block) fsParent() FsNode { return block.parent }

// Represent a file in a hierarchical tree model.
type File struct {
	name   string
	mode   uint32
	strong string
	parent *Dir

	Size   int64
	Blocks []*Block
}

func (file *File) Name() string { return file.name }

func (file *File) Mode() uint32 { return file.mode }

// For our purposes, files are never considered root nodes.
func (file *File) IsRoot() bool { return file.parent == nil }

func (file *File) Strong() string { return file.strong }

func (file *File) Parent() Node { return file.parent }

func (file *File) fsParent() FsNode { return file.parent }

// Represent a directory in a hierarchical tree model.
type Dir struct {
	name   string
	mode   uint32
	strong string
	parent *Dir

	SubDirs []*Dir
	Files   []*File
}

func (dir *Dir) Name() string { return dir.name }

func (dir *Dir) Mode() uint32 { return dir.mode }

func (dir *Dir) IsRoot() bool { return dir.parent == nil }

// Get the directory's strong checksum, based on its deep contents.
// This is calculated in a similar manner to the way git checksums directories.
// Because it is expensive, the value is cached on first access.
func (dir *Dir) Strong() string {
	if dir.strong == "" {
		dir.strong = dir.calcStrong()
	}
	return dir.strong
}

// Calculate the strong checksum of a directory.
func (dir *Dir) calcStrong() string {
	var sha1 = sha1.New()
	sha1.Write(dir.stringBytes())
	return toHexString(sha1)
}

func (dir *Dir) Parent() Node { return dir.parent }

func (dir *Dir) fsParent() FsNode { return dir.parent }

// Represent the directory's distinct deep contents as a byte array.
// Inspired by skimming over git internals.
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
func (dir *Dir) String() string {
	return string(dir.stringBytes())
}

func (dir *Dir) Resolve(relpath string) (fsNode FsNode, hasItem bool) {
	parts := SplitNames(relpath)
	cwd := dir
	i := 0
	l := len(parts)

	for i = 0; i < l; i++ {
		fsNode, hasItem = cwd.Item(parts[i])
		if !hasItem {
			return nil, false
		}

		if i == l-1 {
			return fsNode, true
		}

		switch t := fsNode.(type) {
		case *Dir:
			cwd = fsNode.(*Dir)
		default:
			return nil, false
		}
	}

	return nil, false
}

func (dir *Dir) Item(name string) (FsNode, bool) {
	for _, subdir := range dir.SubDirs {
		if subdir.Name() == name {
			return subdir, true
		}
	}

	for _, file := range dir.Files {
		if file.Name() == name {
			return file, true
		}
	}

	return nil, false
}

// Visitor function to traverse a hierarchical tree model.
type NodeVisitor func(Node) bool

// Traverse the hierarchical tree model with a user-defined NodeVisitor function.
func Walk(node Node, visitor NodeVisitor) {
	nodestack := []Node{}
	nodestack = append(nodestack, node)

	for len(nodestack) > 0 {
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
