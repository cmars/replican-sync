package fs

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

// Provide access to the raw byte storage.
type BlockStore interface {

	// Get the root hierarchical index node
	Root() FsNode

	Index() *BlockIndex

	// Given a strong checksum of a block, get the bytes for that block.
	ReadBlock(strong string) ([]byte, os.Error)

	// Given the strong checksum of a file, start and end positions, get those bytes.
	ReadInto(strong string, from int64, length int64, writer io.Writer) (int64, os.Error)
}

// A local file implementation of BlockStore
type LocalStore interface {
	BlockStore

	RelPath(fullpath string) (relpath string)

	Relocate(fullpath string) (relocFullpath string, err os.Error)

	Resolve(relpath string) string

	Init() os.Error

	reindex() os.Error
}

type LocalInfo struct {
	RootPath string
	Filter   IndexFilter
	index    *BlockIndex
	relocs   map[string]string
}

type LocalDirStore struct {
	*LocalInfo
	dir *Dir
}

type LocalFileStore struct {
	*LocalInfo
	file *File
}

func NewLocalStore(rootPath string) (local LocalStore, err os.Error) {
	rootInfo, err := os.Stat(rootPath)
	if err != nil {
		return nil, err
	}

	localInfo := &LocalInfo{RootPath: rootPath}
	if rootInfo.IsDirectory() {
		local = &LocalDirStore{LocalInfo: localInfo}
	} else if rootInfo.IsRegular() {
		local = &LocalFileStore{LocalInfo: localInfo}
	}

	err = local.Init()
	return local, err
}

func (store *LocalDirStore) Init() os.Error {
	store.relocs = make(map[string]string)
	if store.Filter == nil {
		store.Filter = IncludeAll
	}
	return store.reindex()
}

func (store *LocalFileStore) Init() os.Error {
	store.relocs = make(map[string]string)
	return store.reindex()
}

func (store *LocalDirStore) reindex() (err os.Error) {
	store.dir = IndexDir(store.RootPath, store.Filter, nil)
	if store.dir == nil {
		return os.NewError(fmt.Sprintf("Failed to reindex root: %s", store.RootPath))
	}

	store.index = IndexBlocks(store.dir)
	return nil
}

func (store *LocalFileStore) reindex() (err os.Error) {
	store.file, err = IndexFile(store.RootPath)
	if err != nil {
		return err
	}

	store.index = IndexBlocks(store.file)
	return nil
}

func (store *LocalInfo) RelPath(fullpath string) (relpath string) {
	return MakeRelative(fullpath, store.RootPath)
}

const RELOC_PREFIX string = "_reloc"

func (store *LocalInfo) Relocate(fullpath string) (relocFullpath string, err os.Error) {
	relocFh, err := ioutil.TempFile(store.RootPath, RELOC_PREFIX)
	if err != nil {
		return "", err
	}

	relocFullpath = relocFh.Name()

	err = relocFh.Close()
	if err != nil {
		return "", err
	}

	err = os.Remove(relocFh.Name())
	if err != nil {
		return "", err
	}

	err = Move(fullpath, relocFullpath)
	if err != nil {
		return "", err
	}

	relpath := store.RelPath(fullpath)
	relocRelpath := store.RelPath(relocFullpath)

	store.relocs[relpath] = relocRelpath
	return relocFullpath, nil
}

func (store *LocalInfo) Resolve(relpath string) string {
	if relocPath, hasReloc := store.relocs[relpath]; hasReloc {
		relpath = relocPath
	}

	return filepath.Join(store.RootPath, relpath)
}

func (store *LocalDirStore) Root() FsNode { return store.dir }

func (store *LocalFileStore) Root() FsNode { return store.file }

func (store *LocalInfo) Index() *BlockIndex { return store.index }

func (store *LocalInfo) ReadBlock(strong string) ([]byte, os.Error) {
	block, has := store.index.StrongBlock(strong)
	if !has {
		return nil, os.NewError(
			fmt.Sprintf("Block with strong checksum %s not found", strong))
	}

	buf := &bytes.Buffer{}
	_, err := store.ReadInto(block.Parent().Strong(), block.Offset(), int64(BLOCKSIZE), buf)
	if err == nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (store *LocalInfo) ReadInto(strong string, from int64, length int64, writer io.Writer) (int64, os.Error) {

	file, has := store.index.StrongFile(strong)
	if !has {
		return 0,
			os.NewError(fmt.Sprintf("File with strong checksum %s not found", strong))
	}

	path := store.Resolve(RelPath(file))

	fh, err := os.Open(path)
	if fh == nil {
		return 0, err
	}

	_, err = fh.Seek(from, 0)
	if err != nil {
		return 0, err
	}

	n, err := io.Copyn(writer, fh, length)
	if err != nil {
		return n, err
	}

	return n, nil
}
