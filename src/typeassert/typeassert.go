package main
 
import (
    "fmt"
    "path/filepath"
)

const BLOCKSIZE int = 8192

// Nodes are any member of a hierarchical index.
type Node interface {
	
	// Test if this node is at the root of the index.
	IsRoot() bool
	
	// Get the node that contains this node in the hierarchical index.
	Parent() Node
	
	// Get the nth child node contained by this node.
	Child(i int) Node
	
	// Get the number of child nodes contained by this node.
	ChildCount() int
	
}

// FsNodes are members of a hierarchical index that correlate to the filesystem.
type FsNode interface {
	
	// FsNode extends the concept of Node.
	Node
	
	// FsNodes all have names.
	Name() string
	
}

func RelPath(node FsNode) string {
	parts := []string{}
	
	for fsNode, isFsNode := node.(FsNode); fsNode != nil && isFsNode ; {
		fmt.Printf("%v\n", fsNode)
		
		if fsNode == nil {
			break
		}
		
		if len(parts) > 0 {
			parts = append([]string{fsNode.Name()}, parts[1:]...)
		} else {
			parts = append(parts, fsNode.Name())
		}
		
		parent := fsNode.Parent()
		if parent == nil {
			fmt.Printf("break! break! brakes? nooooooooooo stop!!!\n")
			break
		} else {
			fmt.Printf("%s: my parent %v is not nil. right? right?!\n", fsNode, parent)
		}
		fsNode, isFsNode = parent.(FsNode)
	}
	return filepath.Join(parts...)
}

func RelPath2(node FsNode) string {
	parts := []string{}
	
	for fsNode, isFsNode := node.(FsNode); fsNode != nil && isFsNode ; {
		fmt.Printf("%v\n", fsNode)
		
		if fsNode == nil {
			break
		}
		
		if len(parts) > 0 {
			parts = append([]string{fsNode.Name()}, parts[1:]...)
		} else {
			parts = append(parts, fsNode.Name())
		}
		
		if dir, isDir := fsNode.(*Dir); isDir && dir.parent == nil {
			break
		}
		
		parent := fsNode.Parent()
		if parent == nil {
			fmt.Printf("break! break! brakes? nooooooooooo stop!!!\n")
			break
		} else {
			fmt.Printf("%s: my parent %v is not nil. right? right?!\n", fsNode, parent)
		}
		fsNode, isFsNode = parent.(FsNode)
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

func (block *Block) Parent() (Node) { return block.parent }

func (block *Block) Child(i int) (Node) { return nil }

func (block *Block) ChildCount() (int) { return 0 }

// Represent a file in a hierarchical index.
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

// Represent a directory in a hierarchical index.
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
	return dir.strong
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

func (dir *Dir) ChildCount() (int) { return len(dir.subdirs) + len(dir.files) }

func main() {
	root := &Dir{name:""}
	
	etc := &Dir{name:"etc", parent:root}
	root.subdirs = append(root.subdirs, etc)
	
	passwd := &File{name:"hosts", parent:etc}
	etc.files = append(etc.files, passwd)
	
	for i := 0; i < 3; i++ {
		passwd.blocks = append(passwd.blocks, &Block{position:i})
	}
	
	usr := &Dir{name:"usr", parent:root}
	root.subdirs = append(root.subdirs, usr)
	
	usrbin := &Dir{name:"bin", parent:usr}
	usr.subdirs = append(root.subdirs, usrbin)
	
	ls := &File{name:"ls", parent:usrbin}
	usrbin.files = append(usrbin.files, ls)
	
	for i := 0; i < 5; i++ {
		ls.blocks = append(ls.blocks, &Block{position:i})
	}
	
	fmt.Print("\nUsing RelPath2:\n\n")
	
	fmt.Print(RelPath2(root))
	fmt.Print(RelPath2(etc))
	fmt.Print(RelPath2(passwd))
	fmt.Print(RelPath2(usr))
	fmt.Print(RelPath2(usrbin))
	fmt.Print(RelPath2(ls))
	
	fmt.Print("\nUsing RelPath:\n\n")
	
	fmt.Print(RelPath(root))
	fmt.Print(RelPath(etc))
	fmt.Print(RelPath(passwd))
	fmt.Print(RelPath(usr))
	fmt.Print(RelPath(usrbin))
	fmt.Print(RelPath(ls))
}

