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
		weak.a += b
		weak.b += (len(buf) - i) * b
	}
}

// Get the current weak checksum value
func (weak *WeakChecksum) Get() int {
	return weak.b<<16 | weak.a
}

// Roll the checksum forward by one byte
func (weak *WeakChecksum) Roll(removedByte byte, newByte byte) {
	weak.a -= int(removedByte) - int(newByte)
	weak.b -= int(removedByte)*BLOCKSIZE - weak.a
}

type IndexFilter func(path string, f *os.FileInfo) bool

func AlwaysMatch(path string, f *os.FileInfo) bool { return true }

func AllMatch(filters ...IndexFilter) IndexFilter {
	return func(path string, f *os.FileInfo) bool {
		for _, fn := range filters {
			if !fn(path, f) {
				return false
			}
		}
		return true
	}
}

func AnyMatch(filters ...IndexFilter) IndexFilter {
	return func(path string, f *os.FileInfo) bool {
		for _, fn := range filters {
			if fn(path, f) {
				return true
			}
		}
		return false
	}
}

type Indexer struct {
	Path   string
	Repo   NodeRepo
	Filter IndexFilter
	Errors chan<- os.Error

	root   Dir
	dirMap map[string]Dir
}

// Initialize the Indexer for filepath.Walk visit
func (indexer *Indexer) initWalk() {
	indexer.Path = filepath.Clean(indexer.Path)
	indexer.Path = strings.TrimRight(indexer.Path, "/\\")

	if indexer.Filter == nil {
		indexer.Filter = AlwaysMatch
	}

	indexer.root = nil
	indexer.dirMap = make(map[string]Dir)

	if rootInfo, err := os.Stat(indexer.Path); err == nil {
		indexer.VisitDir(indexer.Path, rootInfo)
		indexer.root = indexer.dirMap[indexer.Path]
	}
}

// Indexer callback for directories
func (indexer *Indexer) VisitDir(path string, f *os.FileInfo) bool {
	if !indexer.Filter(path, f) {
		return false
	}

	path = filepath.Clean(path)
	dir, hasDir := indexer.dirMap[path]
	if !hasDir {
		dirname, basename := filepath.Split(path)
		dirname = strings.TrimRight(dirname, "/\\") // remove the trailing slash

		parentDir, hasParent := indexer.dirMap[dirname]
		info := &DirInfo{
			Name: basename,
			Mode: f.Mode}
		if hasParent {
			info.Parent = parentDir.Info().Strong
			dir = indexer.Repo.AddDir(parentDir, info)
		} else {
			info.Name = ""
			dir = indexer.Repo.AddDir(nil, info)
		}
		indexer.dirMap[path] = dir
	}

	return true
}

// IndexDir visitor callback for files
func (indexer *Indexer) VisitFile(path string, f *os.FileInfo) {
	if !indexer.Filter(path, f) {
		return
	}

	fileInfo, blocksInfo, err := IndexFile(path)
	if err == nil {
		dirpath, _ := filepath.Split(path)
		dirpath = filepath.Clean(dirpath)
		if dirinfo, err := os.Stat(dirpath); err == nil {
			indexer.VisitDir(dirpath, dirinfo)

			if fileParent, hasParent := indexer.dirMap[dirpath]; hasParent {
				indexer.Repo.AddFile(fileParent, fileInfo, blocksInfo)
				return
			} else if indexer.Errors != nil {
				indexer.Errors <- os.NewError("cannot locate parent directory")
			}
		}
	} else if indexer.Errors != nil {
		indexer.Errors <- err
	}
}

func IndexDir(path string, repo NodeRepo) (Dir, []os.Error) {
	errors := []os.Error{}
	dirChan := make(chan Dir, 1)
	errorChan := make(chan os.Error, 1)
	indexer := &Indexer{Path: path, Repo: repo, Errors: errorChan}
	go func() {
		dirChan <- indexer.Index()
		close(errorChan)
	}()
	for error := range errorChan {
		errors = append(errors, error)
	}
	dir := <-dirChan
	return dir, errors
}

func (indexer *Indexer) Index() Dir {
	control := make(chan bool)
	indexer.initWalk()

	go func() {
		filepath.Walk(indexer.Path, indexer, indexer.Errors)
		close(control)
	}()
	<-control

	if indexer.root != nil {
		indexer.root.UpdateStrong()
	}

	return indexer.root
}

// Build a hierarchical tree model representing a file's contents
func IndexFile(path string) (fileInfo *FileInfo, blocksInfo []*BlockInfo, err os.Error) {
	var f *os.File
	var buf [BLOCKSIZE]byte

	stat, err := os.Stat(path)
	if stat == nil {
		return nil, nil, err
	} else if !stat.IsRegular() {
		return nil, nil, os.NewError(fmt.Sprintf("%s: not a regular file", path))
	}

	f, err = os.Open(path)
	if f == nil {
		return nil, nil, err
	}
	defer f.Close()

	_, basename := filepath.Split(path)
	fileInfo = &FileInfo{
		Name: basename,
		Mode: stat.Mode,
		Size: stat.Size}

	var block *BlockInfo
	sha1 := sha1.New()
	blockNum := 0
	blocksInfo = []*BlockInfo{}

	for {
		switch rd, err := f.Read(buf[:]); true {
		case rd < 0:
			return nil, nil, err
		case rd == 0:
			fileInfo.Strong = toHexString(sha1)
			return fileInfo, blocksInfo, nil
		case rd > 0:
			// Update block hashes
			block = IndexBlock(buf[0:rd])
			block.Position = blockNum
			blocksInfo = append(blocksInfo, block)

			// update file hash
			sha1.Write(buf[0:rd])

			// Increment block counter
			blockNum++
		}
	}
	panic("Impossible")
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
func IndexBlock(buf []byte) *BlockInfo {
	var weak = new(WeakChecksum)
	weak.Write(buf)

	return &BlockInfo{
		Weak:   weak.Get(),
		Strong: StrongChecksum(buf)}
}
