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
	scanner RecSource
	cask *gocask.Gocask
}

func NewCaskWriter(scanner RecSource, cask* gocask.Gocask) *CaskWriter {
	return &CaskWriter{
		scanner: scanner,
		cask: cask }
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

func (caskWriter *CaskWriter) WriteAll() {
	var err error
	for {
		rec, ok := <- caskWriter.scanner.Records()
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
		if !ok { return }
	}
}
