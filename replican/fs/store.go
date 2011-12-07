package fs

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// Provide access to the raw byte storage.
type BlockStore interface {
	Repo() NodeRepo

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

	RootPath() string

	reindex() os.Error
}

type localBase struct {
	rootPath string
	repo     NodeRepo
	relocs   map[string]string
}

type LocalDirStore struct {
	*localBase
	dir Dir
}

type LocalFileStore struct {
	*localBase
	file File
}

func NewLocalStore(rootPath string, repo NodeRepo) (local LocalStore, err os.Error) {
	rootInfo, err := os.Stat(rootPath)
	if err != nil {
		return nil, err
	}

	localBase := &localBase{rootPath: rootPath, repo: repo}
	if rootInfo.IsDirectory() {
		local = &LocalDirStore{localBase: localBase}
	} else if rootInfo.IsRegular() {
		local = &LocalFileStore{localBase: localBase}
	}

	localBase.relocs = make(map[string]string)

	if err := local.reindex(); err != nil {
		return nil, err
	}

	return local, nil
}

func (store *LocalDirStore) reindex() (err os.Error) {
	store.dir = IndexDir(store.RootPath(), store.repo, nil)
	if store.dir == nil {
		return os.NewError(fmt.Sprintf("Failed to reindex root: %s", store.RootPath()))
	}

	return nil
}

func (store *LocalFileStore) reindex() (err os.Error) {
	store.file, err = IndexFile(store.RootPath(), store.repo)
	if err != nil {
		return err
	}

	return nil
}

func (store *localBase) RelPath(fullpath string) (relpath string) {
	relpath = strings.Replace(fullpath, store.RootPath(), "", 1)
	relpath = strings.TrimLeft(relpath, "/\\")
	return relpath
}

const RELOC_PREFIX string = "_reloc"

func (store *localBase) Relocate(fullpath string) (relocFullpath string, err os.Error) {
	relocFh, err := ioutil.TempFile(store.RootPath(), RELOC_PREFIX)
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

func (store *localBase) Resolve(relpath string) string {
	if relocPath, hasReloc := store.relocs[relpath]; hasReloc {
		relpath = relocPath
	}

	return filepath.Join(store.RootPath(), relpath)
}

func (store *localBase) RootPath() string { return store.rootPath }

func (store *localBase) Repo() NodeRepo { return store.repo }

func (store *LocalDirStore) Root() FsNode { return store.dir }

func (store *LocalFileStore) Root() FsNode { return store.file }

func (store *localBase) ReadBlock(strong string) ([]byte, os.Error) {
	block, has := store.repo.Block(strong)
	if !has {
		return nil, os.NewError(
			fmt.Sprintf("Block with strong checksum %s not found", strong))
	}

	buf := &bytes.Buffer{}
	_, err := store.ReadInto(block.Info().Strong, block.Info().Offset(), int64(BLOCKSIZE), buf)
	if err == nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (store *localBase) ReadInto(strong string, from int64, length int64, writer io.Writer) (int64, os.Error) {

	file, has := store.repo.File(strong)
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
