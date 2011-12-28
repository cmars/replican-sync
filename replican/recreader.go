package replican

import (
	"bytes"
	"encoding/binary"
	"io"
	"log"
)

type RecReader struct {
	outputs []chan *ScanRec
	reader io.Reader
}

func NewRecReader(reader io.Reader) *RecReader {
	recReader := &RecReader{
		reader: reader,
		outputs: []chan *ScanRec{} }
	return recReader
}

func (recReader *RecReader) AddOutput(records chan *ScanRec) {
	recReader.outputs = append(recReader.outputs, records)
}

func (recReader *RecReader) Start() {
	go recReader.run()
}

func (recReader *RecReader) run() {
	recPos := int32(0)
	for {
		rec := &ScanRec{ Seq: recPos }
		buf := make([]byte, RECSIZE)
		n, err := io.ReadFull(recReader.reader, buf)
		if err != nil {
			if (err == io.ErrUnexpectedEOF && n > 0) {
				log.Printf("read failed: only %d bytes read, %v", n, err)
			}
			break
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
			log.Printf("invalid record: %v", rec)
			rec = nil
		}
		
		if err != nil {
			log.Printf("read failed: %v", err)
		}
		
		if rec != nil {
			sendRec(recReader.outputs, rec)
			recPos++
		}
	}
	
	closeAll(recReader.outputs)
}
