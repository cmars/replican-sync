
package blocks

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

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


