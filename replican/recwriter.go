package replican

import (
	"encoding/binary"
	"io"
	"log"
)

type RecWriter struct {
	records chan *ScanRec
	writer io.Writer
	exitChan chan bool
}

func NewRecWriter(records chan *ScanRec, writer io.Writer) *RecWriter {
	return &RecWriter{
		records: records,
		writer: writer,
		exitChan: make(chan bool) }
}

func (recWriter *RecWriter) Start() {
	go recWriter.run()
}

func (recWriter *RecWriter) Wait() {
	_ = <- recWriter.exitChan
}

func (recWriter *RecWriter) run() {
	var err error
	for {
		rec, ok := <- recWriter.records
		if rec != nil {
//			log.Printf("write: %v", rec)
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
		if !ok { break }
	}
	
	close(recWriter.exitChan)
}
