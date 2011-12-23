package replican

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// Message containing a scanned record during a directory walk.
// Only one of Block, File or Dir will be set, the others will be nil.
type ScanRec struct {

	// The sequence number is the order in which the record occurs.
	// This alone can be used to locate the record in the stream since records 
	// have a fixed size.
	Seq int
	
	// The path associated with the record.
	// If the record is a block, the path is an empty string.
	Path string
	
	Block *BlockRec
	File *FileRec
	Dir *DirRec
}

type RecSource interface {
	
	Records() <- chan *ScanRec
	
}

type Walker struct {
	path string
	records chan *ScanRec
}

func NewWalker(path string) *Walker {
	walker := &Walker{
		path: path,
		records: make(chan *ScanRec) }
	go walker.run()
	return walker
}

func (walker *Walker) Records() <- chan *ScanRec { return walker.records }

func (walker *Walker) run() {
	recPos := int(0)
	lastSibling := make(map[int]int) // recPos of last directory at given depth
	dirbufs := make(map[int]*bytes.Buffer) // directory entry in progress at level

	PostOrderWalk(walker.path, func(path string, info os.FileInfo, err error) error {
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
					Sibling: int32(sibling),
					Depth:   int32(depth)}
			
			parentDirent = fmt.Sprintf("d\t%x\t%s\n", dirStrong, parts[len(parts)-1])
			
			delete(lastSibling, depth+1)
			lastSibling[depth] = recPos

			walker.records <- &ScanRec{
				Seq: recPos,
				Path: path,
				Dir: dirRec }
			
			recPos++
		} else {
			if fileRec, blocksRec, err := ScanBlocks(path); err == nil {
				
				sibling, has := lastSibling[depth]
				if !has { sibling = -1 }
				fileRec.Sibling = int32(sibling)
				
				for _, blockRec := range blocksRec {
					walker.records <- &ScanRec{
						Seq: recPos,
						Block: blockRec }
					recPos++
				}
				
				parentDirent = fmt.Sprintf("f\t%x\t%s\n", fileRec.Strong, parts[len(parts)-1])
				
				walker.records <- &ScanRec{
					Seq: recPos,
					Path: path,
					File: fileRec }
				
				lastSibling[depth] = recPos
				recPos++
				
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
	
	close(walker.records)
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
			block.Position = int32(blockNum)
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

	rec := &BlockRec{ Type: BLOCK, Weak: int32(weak.Get()) }
	copyStrong(hash.Sum(nil), &rec.Strong)
	return rec
}

func splitNames(path string) []string {
	if path == "" {
		return []string{}
	}
	return strings.Split(path, string(os.PathSeparator))
}

type RecWriter struct {
	scanner RecSource
	writer io.Writer
}

func NewRecWriter(scanner RecSource, writer io.Writer) *RecWriter {
	recWriter := &RecWriter{
		scanner: scanner,
		writer: writer }
	return recWriter
}

func (recWriter *RecWriter) WriteAll() {
	var err error
	for {
		rec, ok := <- recWriter.scanner.Records()
		if rec != nil {
			switch {
			case rec.Block != nil:
				err = binary.Write(recWriter.writer, binary.LittleEndian, rec.Block)
			case rec.File != nil:
				err = binary.Write(recWriter.writer, binary.LittleEndian, rec.File)
			case rec.Dir != nil:
				err = binary.Write(recWriter.writer, binary.LittleEndian, rec.Dir)
			default:
				log.Printf("empty record: %v", rec)
			}
			
			if err != nil {
				log.Printf("write %v failed: %v", rec, err)
			}
		}
		if !ok { return }
	}
}

type RecReader struct {
	records chan *ScanRec
	reader io.Reader
}

func NewRecReader(reader io.Reader) *RecReader {
	recReader := &RecReader{
		reader: reader,
		records: make(chan *ScanRec) }
	go recReader.run()
	return recReader
}

func (recReader *RecReader) Records() <- chan *ScanRec {
	return recReader.records
}

func (recReader *RecReader) run() {
	var rec *ScanRec
	recPos := 0
	for {
		rec = &ScanRec{ Seq: recPos }
		buf := make([]byte, RECSIZE)
		n, err := io.ReadFull(recReader.reader, buf)
		if err != nil {
			if (err == io.ErrUnexpectedEOF && n > 0) {
				log.Printf("read failed: only %d bytes read, %v", n, err)
			}
			break
		}
		recPos++
//		log.Printf("buf: %x", buf)
		bbuf := bytes.NewBuffer(buf)
		switch RecType(buf[0]) {
		case BLOCK:
			rec.Block = new(BlockRec)
			err = binary.Read(bbuf, binary.LittleEndian, rec.Block)
		case FILE:
			rec.File = new(FileRec)
			err = binary.Read(bbuf, binary.LittleEndian, rec.File)
		case DIR:
			rec.Dir = new(DirRec)
			err = binary.Read(bbuf, binary.LittleEndian, rec.Dir)
		default:
			log.Printf("invalid record: %v", rec)
			rec = nil
		}
		
		if err != nil {
			log.Printf("read failed: %v", err)
		}
		
		if rec != nil {
			recReader.records <- rec
		}
	}
	close(recReader.records)
}
