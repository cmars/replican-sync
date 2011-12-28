package replican

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"path/filepath"
	
	gocask "gocask.googlecode.com/hg"
)

type CaskWriter struct {
	records chan *ScanRec
	cask *gocask.Gocask
	exitChan chan bool
}

func NewCaskWriter(records chan *ScanRec, cask* gocask.Gocask) *CaskWriter {
	return &CaskWriter{
		records: records,
		cask: cask,
		exitChan: make(chan bool) }
}

func intToBytes(i int32) []byte {
	buf := bytes.NewBuffer([]byte{})
	binary.Write(buf, binary.LittleEndian, &i)
	return buf.Bytes()
}

func bytesToInt(b []byte) int32 {
	var i int32
	buf := bytes.NewBuffer(b)
	binary.Read(buf, binary.LittleEndian, &i)
	return i
}

func (caskWriter *CaskWriter) Start() {
	go caskWriter.run()
}

func (caskWriter *CaskWriter) Wait() {
	_ = <- caskWriter.exitChan
}

func (caskWriter *CaskWriter) run() {
	var err error
	for {
		rec, ok := <- caskWriter.records
		if rec != nil && rec.Path != "" {
			_, name := filepath.Split(rec.Path)
			seqBytes := intToBytes(rec.Seq)
			caskWriter.cask.Put(fmt.Sprintf("%x", rec.Seq), []byte(name))
			switch {
			case rec.File != nil:
				err = caskWriter.cask.Put(fmt.Sprintf("%x", rec.File.Strong), seqBytes)
			case rec.Dir != nil:
				err = caskWriter.cask.Put(fmt.Sprintf("%x", rec.Dir.Strong), seqBytes)
			}
			
			if err != nil {
				log.Printf("put %v failed: %v", rec, err)
			}
		}
		if !ok { break }
	}
	close(caskWriter.exitChan)
}
