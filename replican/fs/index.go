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

// Visitor used to traverse a directory with filepath.Walk in IndexDir
type indexVisitor struct {
	root   Dir
	repo   NodeRepo
	dirMap map[string]Dir
	errors chan<- os.Error
}

// Initialize the IndexDir visitor
func newVisitor(path string, repo NodeRepo) *indexVisitor {
	path = filepath.Clean(path)
	path = strings.TrimRight(path, "/\\")

	visitor := &indexVisitor{
		errors: make(chan os.Error),
		dirMap: make(map[string]Dir),
		repo:   repo}

	if rootInfo, err := os.Stat(path); err == nil {
		visitor.VisitDir(path, rootInfo)
		visitor.root = visitor.dirMap[path]
		visitor.root.Info().Name = ""
	}

	return visitor
}

// IndexDir visitor callback for directories
func (visitor *indexVisitor) VisitDir(path string, f *os.FileInfo) bool {
	path = filepath.Clean(path)
	dir, hasDir := visitor.dirMap[path]
	if !hasDir {
		dirname, basename := filepath.Split(path)
		dirname = strings.TrimRight(dirname, "/\\") // remove the trailing slash

		parentDir, hasParent := visitor.dirMap[dirname]
		info := &DirInfo{
			Name: basename,
			Mode: f.Mode}
		if hasParent {
			info.Parent = parentDir.Info().Strong
		}
		dir = visitor.repo.AddDir(parentDir, info)
		visitor.dirMap[path] = dir
	}

	return true
}

// IndexDir visitor callback for files
func (visitor *indexVisitor) VisitFile(path string, f *os.FileInfo) {
	fileInfo, blocksInfo, err := IndexFile(path)
	if err == nil {
		dirpath, _ := filepath.Split(path)
		dirpath = filepath.Clean(dirpath)
		if dirinfo, err := os.Stat(dirpath); err == nil {
			visitor.VisitDir(dirpath, dirinfo)

			if fileParent, hasParent := visitor.dirMap[dirpath]; hasParent {
				visitor.repo.AddFile(fileParent, fileInfo, blocksInfo)
				return
			} else if visitor.errors != nil {
				visitor.errors <- os.NewError("cannot locate parent directory")
			}
		}
	} else if visitor.errors != nil {
		visitor.errors <- err
	}
}

// Build a hierarchical tree model representing a directory's contents
func IndexDir(path string, repo NodeRepo, errors chan<- os.Error) Dir {
	control := make(chan bool)
	visitor := newVisitor(path, repo)
	visitor.errors = errors

	go func() {
		filepath.Walk(path, visitor, errors)
		close(control)
	}()
	<-control

	visitor.root.UpdateStrong()

	return visitor.root
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
