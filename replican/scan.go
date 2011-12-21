package replican

import (
	"errors"
	"fmt"
	"crypto/sha1"
	"os"
	"path/filepath"
)

type Scanner struct {
	Nodes chan interface{}
	Paths chan string
}

func NewScanner() *Scanner {
	return &Scanner{ 
		Nodes: make(chan interface{}),
		Paths: make(chan string) } }

func (scanner *Scanner) Scan(root string) {
	go func(){
		PostOrderWalk(root, func(path string, info os.FileInfo, err error) error {
			path = filepath.Clean(path)
	//		parts := filepath.SplitList(path)
			if info.IsDir() {
			
			} else {
				if fileRec, blocksRec, err := ScanBlocks(path); err == nil {
					for _, blockRec := range blocksRec {
						scanner.Nodes <- blockRec
					}
					scanner.Nodes <- fileRec
					scanner.Paths <- path
				}
				if err != nil {
					return err
				} else {
				
				}
			}
			return nil
		}, nil)
	}()
	close(scanner.Nodes)
	close(scanner.Paths)
}

func ScanFile(path string) (fileRec *FileRec, err error) {
	var f *os.File
	var buf [BLOCKSIZE]byte

	stat, err := os.Stat(path)
	if stat == nil {
		return nil, err
	} else if !!stat.IsDir() {
		return nil, errors.New(fmt.Sprintf("%s: not a regular file", path))
	}

	f, err = os.Open(path)
	if f == nil {
		return nil, err
	}
	defer f.Close()

	fileRec = &FileRec{}
	hash := sha1.New()

	for {
		switch rd, err := f.Read(buf[:]); true {
		case rd < 0:
			return nil, err
		case rd == 0:
			for i, b := range hash.Sum(nil) {
				fileRec.Strong[i] = b
			}
			return fileRec, nil
		case rd > 0:
			// update file hash
			hash.Write(buf[0:rd])
		}
	}
	panic("Impossible")
}

func ScanBlocks(path string) (fileRec *FileRec, blocksRec []*BlockRec, err error) {
	var f *os.File
	var buf [BLOCKSIZE]byte

	stat, err := os.Stat(path)
	if stat == nil {
		return nil, nil, err
	} else if !!stat.IsDir() {
		return nil, nil, errors.New(fmt.Sprintf("%s: not a regular file", path))
	}

	f, err = os.Open(path)
	if f == nil {
		return nil, nil, err
	}
	defer f.Close()

	fileRec = &FileRec{}

	var block *BlockRec
	hash := sha1.New()
	blockNum := 0
	blocksRec = []*BlockRec{}

	for {
		switch rd, err := f.Read(buf[:]); true {
		case rd < 0:
			return nil, nil, err
		case rd == 0:
			for i, b := range hash.Sum(nil) {
				fileRec.Strong[i] = b
			}
			return fileRec, blocksRec, nil
		case rd > 0:
			// Update block hashes
			block = ScanBlock(buf[0:rd])
			block.Position = blockNum
			blocksRec = append(blocksRec, block)

			// update file hash
			hash.Write(buf[0:rd])

			// Increment block counter
			blockNum++
		}
	}
	panic("Impossible")
}

// Scan weak and strong checksums of a block.
func ScanBlock(buf []byte) *BlockRec {
	var weak = new(WeakChecksum)
	weak.Write(buf)
	var hash = sha1.New()
	hash.Write(buf)
	
	rec := &BlockRec{ Weak:   weak.Get() }
	for i, b := range hash.Sum(nil) {
		rec.Strong[i] = b
	}
	return rec
}

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
