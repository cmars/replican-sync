package replican

import (
	"bytes"
	"encoding/binary"
	"io"
	"log"
)

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
		rec = &ScanRec{ Seq: int32(recPos) }
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
