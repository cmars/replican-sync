package replican

import (
	"encoding/binary"
	"io"
	"log"
)

type RecWriter struct {
	scanner RecSource
	writer io.Writer
	records chan *ScanRec
}

func NewRecWriter(scanner RecSource, writer io.Writer) *RecWriter {
	recWriter := &RecWriter{
		scanner: scanner,
		writer: writer }
	return recWriter
}

func (recWriter *RecWriter) Records() <- chan *ScanRec { return recWriter.records }

func (recWriter *RecWriter) WriteAll() {
	recWriter.WriteFwd(nil)
}

func (recWriter *RecWriter) WriteFwd(records chan *ScanRec) {
	if records != nil {
		recWriter.records = records
	}
	
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
			
			if recWriter.records != nil {
				recWriter.records <- rec
			}
		}
		if !ok { return }
	}
}
