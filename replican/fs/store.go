package fs

import (
	"bytes"
	"errors"
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
	ReadBlock(strong string) ([]byte, error)

	// Given the strong checksum of a file, start and end positions, get those bytes.
	ReadInto(strong string, from int64, length int64, writer io.Writer) (int64, error)
}

// A local file implementation of BlockStore
type LocalStore interface {
	BlockStore

	RelPath(fullpath string) (relpath string)

	Relocate(fullpath string) (relocFullpath string, err error)

	Resolve(relpath string) string

	RootPath() string

	reindex() error
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

func NewLocalStore(rootPath string, repo NodeRepo) (local LocalStore, err error) {
	rootInfo, err := os.Stat(rootPath)
	if err != nil {
		return nil, err
	}

	localBase := &localBase{rootPath: rootPath, repo: repo}
	if rootInfo.IsDir() {
		local = &LocalDirStore{localBase: localBase}
	} else if !rootInfo.IsDir() {
		local = &LocalFileStore{localBase: localBase}
	}

	localBase.relocs = make(map[string]string)

	if err := local.reindex(); err != nil {
		return nil, err
	}

	return local, nil
}

func (store *LocalDirStore) reindex() (err error) {
	indexer := &Indexer{
		Path:   store.RootPath(),
		Repo:   store.repo,
		Filter: store.repo.IndexFilter()}
	store.dir = indexer.Index()
	if store.dir == nil {
		return errors.New(fmt.Sprintf("Failed to reindex root: %s", store.RootPath()))
	}

	return nil
}

func (store *LocalFileStore) reindex() (err error) {
	if fileInfo, blocksInfo, err := IndexFile(store.RootPath()); err == nil {
		store.file = store.repo.AddFile(nil, fileInfo, blocksInfo)
		return nil
	}
	return err
}

func (store *localBase) RelPath(fullpath string) (relpath string) {
	relpath = strings.Replace(fullpath, store.RootPath(), "", 1)
	relpath = strings.TrimLeft(relpath, "/\\")
	return relpath
}

const RELOC_PREFIX string = "_reloc"

func (store *localBase) Relocate(fullpath string) (relocFullpath string, err error) {
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

func (store *LocalFileStore) Resolve(_ string) string {
	return store.RootPath()
}

func (store *localBase) RootPath() string { return store.rootPath }

func (store *localBase) Repo() NodeRepo { return store.repo }

func (store *LocalDirStore) Root() FsNode { return store.dir }

func (store *LocalFileStore) Root() FsNode { return store.file }

func (store *localBase) ReadBlock(strong string) ([]byte, error) {
	block, has := store.repo.Block(strong)
	if !has {
		return nil, errors.New(
			fmt.Sprintf("Block with strong checksum %s not found", strong))
	}

	buf := &bytes.Buffer{}
	_, err := store.ReadInto(block.Info().Strong, block.Info().Offset(), int64(BLOCKSIZE), buf)
	if err == nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (store *localBase) ReadInto(strong string, from int64, length int64, writer io.Writer) (int64, error) {

	file, has := store.repo.File(strong)
	if !has {
		return 0,
			errors.New(fmt.Sprintf("File with strong checksum %s not found", strong))
	}

	path := store.Resolve(RelPath(file))
	return store.readInto(path, from, length, writer)
}

func (store *LocalFileStore) ReadInto(strong string, from int64, length int64, writer io.Writer) (int64, error) {

	file, has := store.repo.File(strong)
	if !has {
		return 0,
			errors.New(fmt.Sprintf("File with strong checksum %s not found", strong))
	}

	path := store.Resolve(RelPath(file))
	return store.readInto(path, from, length, writer)
}

func (store *localBase) readInto(path string, from int64, length int64, writer io.Writer) (int64, error) {
	fh, err := os.Open(path)
	if fh == nil {
		return 0, err
	}

	_, err = fh.Seek(from, 0)
	if err != nil {
		return 0, err
	}

	n, err := io.CopyN(writer, fh, length)
	if err != nil {
		return n, err
	}

	return n, nil
}
