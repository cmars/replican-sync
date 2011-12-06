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

		dir = visitor.repo.CreateDir(&DirInfo{
			Name:   basename,
			Mode:   f.Mode,
			Parent: visitor.dirMap[dirname].Info().Strong})
		visitor.dirMap[path] = dir
		visitor.repo.SetDir(dir)
	}

	return true
}

// IndexDir visitor callback for files
func (visitor *indexVisitor) VisitFile(path string, f *os.FileInfo) {
	file, err := IndexFile(path, visitor.repo)
	if file != nil {
		dirpath, _ := filepath.Split(path)
		dirpath = filepath.Clean(dirpath)
		if dirinfo, err := os.Stat(dirpath); err == nil {
			visitor.VisitDir(dirpath, dirinfo)

			var hasParent bool
			if fileParent, hasParent := visitor.dirMap[dirpath]; hasParent {
				file.Info().Parent = fileParent.Info().Strong
				visitor.repo.SetFile(file)
				return
			} else if visitor.errors != nil {
				visitor.errors <- os.NewError("cannot locate parent directory")
			}
		}
	}

	if err != nil && visitor.errors != nil {
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

	/*
		if visitor.root != nil {
			visitor.root.Info().Strong()
		}
	*/

	return visitor.root
}

// Build a hierarchical tree model representing a file's contents
func IndexFile(path string, repo NodeRepo) (file File, err os.Error) {
	var f *os.File
	var buf [BLOCKSIZE]byte

	stat, err := os.Stat(path)
	if stat == nil {
		return nil, err
	} else if !stat.IsRegular() {
		return nil, os.NewError(fmt.Sprintf("%s: not a regular file", path))
	}

	f, err = os.Open(path)
	if f == nil {
		return nil, err
	}
	defer f.Close()

	_, basename := filepath.Split(path)
	file = repo.CreateFile(&FileInfo{
		Name: basename,
		Mode: stat.Mode,
		Size: stat.Size})

	var block Block
	sha1 := sha1.New()
	blockNum := 0
	blocks := []Block{}

	for {
		switch rd, err := f.Read(buf[:]); true {
		case rd < 0:
			return nil, err
		case rd == 0:
			file.Info().Strong = toHexString(sha1)
			repo.SetFile(file)
			for _, block := range blocks {
				block.Info().Parent = file.Info().Strong
				repo.SetBlock(block)
			}
			return file, nil
		case rd > 0:
			// Update block hashes
			block = IndexBlock(buf[0:rd], repo)
			block.Info().Position = blockNum
			block.Info().Parent = file.Info().Strong
			blocks = append(blocks, block)

			// update file hash
			sha1.Write(buf[0:rd])

			// Increment block counter
			blockNum++
		}
	}

	return nil, nil
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
func IndexBlock(buf []byte, repo NodeRepo) (block Block) {
	var weak = new(WeakChecksum)
	weak.Write(buf)

	block = repo.CreateBlock(&BlockInfo{
		Weak:   weak.Get(),
		Strong: StrongChecksum(buf)})

	return block
}
