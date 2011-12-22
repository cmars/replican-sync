package replican

import (
	"bytes"
	"crypto/sha1"
	"errors"
	"fmt"
//	"log"
	"os"
	"path/filepath"
	"strings"
)

type Scanner struct {
	Records chan interface{}
	Paths chan string
	Done chan bool
}

func NewScanner() *Scanner {
	return &Scanner{
		Records: make(chan interface{}),
		Paths: make(chan string),
		Done: make(chan bool) }
}

func (scanner *Scanner) Start(root string) {
	go func() {
		recPos := 0
		lastSibling := make(map[int]int) // recPos of last directory at given depth
		dirbufs := make(map[int]*bytes.Buffer) // directory entry in progress at level

		PostOrderWalk(root, func(path string, info os.FileInfo, err error) error {
			path = filepath.Clean(path)
			parts := splitNames(path)
//			log.Printf("parts: %v", parts)
			depth := len(parts)
			parentDirent := ""
			
			if info.IsDir() {
				var dirStrong [sha1.Size]byte
				if dirbuf, has := dirbufs[depth+1]; has {
//					log.Printf("dirbuf for %s: |%s|", path, dirbuf.String())
					delete(dirbufs, depth+1)
					dirStrong = StrongChecksum(dirbuf.Bytes())
				} else { // empty directory!
					dirStrong = StrongChecksum([]byte{})
				}
				
				sibling, has := lastSibling[depth]
				if !has { sibling = -1 }
				
				dirRec := &DirRec{
					Type: DIR,
					Strong:  dirStrong,
					Sibling: sibling,
					Depth:   depth}
				
				parentDirent = fmt.Sprintf("d\t%x\t%s\n", dirStrong, parts[len(parts)-1])
				
				delete(lastSibling, depth+1)
				lastSibling[depth] = recPos
				recPos++

				scanner.Records <- dirRec
				scanner.Paths <- path
			} else {
				if fileRec, blocksRec, err := ScanBlocks(path); err == nil {
					
					sibling, has := lastSibling[depth]
					if !has { sibling = -1 }
					fileRec.Sibling = sibling
					
					for _, blockRec := range blocksRec {
						scanner.Records <- blockRec
						recPos++
					}
					
					parentDirent = fmt.Sprintf("f\t%x\t%s\n", fileRec.Strong, parts[len(parts)-1])
					
					lastSibling[depth] = recPos
					recPos++
					
					scanner.Records <- fileRec
					scanner.Paths <- path
					
				} else {
					return err
				}
			}
			
			if parentDirent != "" {
				dirbuf, has := dirbufs[depth]
				if !has {
					dirbuf = bytes.NewBuffer([]byte{})
					dirbufs[depth] = dirbuf
				}
				dirbuf.WriteString(parentDirent)
			}
			
			return nil
		}, nil)
		
		scanner.Done <- true
	}()
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

	fileRec = &FileRec{ Type: FILE }
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

	fileRec = &FileRec{ Type: FILE }

	var block *BlockRec
	hash := sha1.New()
	blockNum := 0
	blocksRec = []*BlockRec{}

	for {
		switch rd, err := f.Read(buf[:]); true {
		case rd < 0:
			return nil, nil, err
		case rd == 0:
			copyStrong(hash.Sum(nil), &fileRec.Strong)
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

	rec := &BlockRec{ Type: BLOCK, Weak: weak.Get() }
	copyStrong(hash.Sum(nil), &rec.Strong)
	return rec
}

func splitNames(path string) []string {
	if path == "" {
		return []string{}
	}
	return strings.Split(path, string(os.PathSeparator))
}
