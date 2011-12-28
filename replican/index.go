package replican

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	
	gocask "gocask.googlecode.com/hg"
)

const INDEX_DIR = ".replican"
const TREE_FILE = "tree"
const CASK_DIR = "cask"

type Index struct {
	path string
	treeFile *os.File
	treeInfo os.FileInfo
	cask *gocask.Gocask
}

func WriteIndex(path string) (*Index, error) {
	walker := NewWalker(path)
	
	caskPath := filepath.Join(path, INDEX_DIR, CASK_DIR)
	os.MkdirAll(caskPath, 0755)
	
	treeFile, err := os.Create(filepath.Join(path, INDEX_DIR, TREE_FILE))
	if err != nil {
		return nil, err
	}
	treeInfo, err := treeFile.Stat()
	if err != nil {
		return nil, err
	}
	
	treeChan := make(chan *ScanRec)
	recWriter := NewRecWriter(treeChan, treeFile)
	walker.AddOutput(treeChan)
	
	caskChan := make(chan *ScanRec)
	cask, err := gocask.NewGocask(caskPath)
	if err != nil {
		return nil, err
	}
	defer cask.Close()
	
	caskWriter := NewCaskWriter(caskChan, cask)
	walker.AddOutput(caskChan)
	
	walker.Start()
	recWriter.Start()
	caskWriter.Start()
	
	recWriter.Wait()
	caskWriter.Wait()
	
	return &Index{ 
		path: path, 
		treeFile: treeFile,
		treeInfo: treeInfo,
		cask: cask }, nil
}

func (index *Index) Close() {
	index.treeFile.Close()
	index.treeFile = nil
	index.cask.Close()
	index.cask = nil
}

func OpenIndex(path string) (*Index, error) {
	treeFile, err := os.Open(filepath.Join(path, INDEX_DIR, TREE_FILE))
	if err != nil {
		return nil, err
	}
	treeInfo, err := treeFile.Stat()
	if err != nil {
		return nil, err
	}
	
	caskPath := filepath.Join(path, INDEX_DIR, CASK_DIR)
	cask, err := gocask.NewGocask(caskPath)
	if err != nil {
		return nil, err
	}
	
	return &Index{
		path: path,
		treeFile: treeFile,
		treeInfo: treeInfo,
		cask: cask }, nil
}

type IndexRec struct {
	Seq int
	index *Index
	
	Block *BlockRec
	File *FileRec
	Dir *DirRec
}

func (index *Index) Root() (*IndexRec, error) {
	return index.RecAt((int(index.treeInfo.Size()) / RECSIZE) - 1)
}

func (index *Index) RecAt(pos int) (*IndexRec, error) {
	rec := &IndexRec{ Seq: pos, index: index }
	buf := make([]byte, RECSIZE)
	_, err := io.ReadFull(index.treeFile, buf)
	if err != nil {
		return nil, err
	}
//		log.Printf("buf: %x", buf)
	bbuf := bytes.NewBuffer(buf)
	switch RecType(buf[0]) {
	case BLOCK:
		rec.Block = new(BlockRec)
		err = binary.Read(bbuf, binary.LittleEndian, rec.Block)
//			log.Printf("read: %v %v", rec, rec.Block)
	case FILE:
		rec.File = new(FileRec)
		err = binary.Read(bbuf, binary.LittleEndian, rec.File)
//			log.Printf("read: %v %v", rec, rec.File)
	case DIR:
		rec.Dir = new(DirRec)
		err = binary.Read(bbuf, binary.LittleEndian, rec.Dir)
//			log.Printf("read: %v %v", rec, rec.Dir)
	default:
		return nil, errors.New(fmt.Sprintf("invalid record: %v", rec))
	}
	return rec, err
}
