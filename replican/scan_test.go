package replican

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/cmars/replican-sync/replican/treegen"

	"github.com/bmizerany/assert"
	
	gocask "gocask.googlecode.com/hg"
)

func makeTree(t *testing.T) string {
	tg := treegen.New()
	treeSpec := tg.D("F",
		tg.D("B",
			tg.F("A", tg.B(1, 100)),
			tg.D("D",
				tg.F("C", tg.B(2, 100)),
				tg.F("E", tg.B(3, 100)))),
		tg.D("G",
			tg.D("I",
				tg.F("H", tg.B(4, 100)))))
	root := treegen.TestTree(t, treeSpec)
	return root
}

func TestScanSiblings(t *testing.T) {
	root := makeTree(t)
	defer os.RemoveAll(root)
	
	walker := NewWalker(root)
	
	recChan := make(chan *ScanRec)
	recs := []*ScanRec{}
	
	l := log.New(ioutil.Discard, "", log.LstdFlags | log.Lshortfile)
//	l = log.New(os.Stdout, "", log.LstdFlags | log.Lshortfile)
	
	walker.AddOutput(recChan)
	walker.Start()
	
	for scanRec := range recChan {
		switch {
		case scanRec.Block != nil:
			l.Printf("%v %v", scanRec.Seq, scanRec.Block.Type)
		case scanRec.File != nil:
			l.Printf("%v %v %x has sibling %v", 
				scanRec.Seq, scanRec.File.Type, 
				scanRec.File.Strong, scanRec.File.Sibling)
		case scanRec.Dir != nil:
			l.Printf("%v %v %x has sibling %v", 
				scanRec.Seq, scanRec.Dir.Type, 
				scanRec.Dir.Strong, scanRec.Dir.Sibling)
		}
		if scanRec.Path != "" {
			l.Printf("Got path: %v", scanRec.Path)
		}
		recs = append(recs, scanRec)
	}
	
//	l.Printf("there are %d records", len(recs))
	
	expectedSiblings := []int{
		-2,	//0 BLOCK
		-1,	//1 FILE /C
		-2, //2 BLOCK
		1,	//3 FILE /E
		-1, //4 DIR /D
		-2,	//5 BLOCK
		4,	//6 FILE /A
		-1, //7 DIR /B
		-2, //8 BLOCK
		-1, //9 /H
		-1, //10 /I
		7,	//11 /G
		-1}	//12 /F
	expectedPaths := []string{
		"",
		"/F/B/D/C",
		"",
		"/F/B/D/E",
		"/F/B/D",
		"",
		"/F/B/A",
		"/F/B",
		"",
		"/F/G/I/H",
		"/F/G/I",
		"/F/G",
		"/F" }
	expectedContents := []string{
		"",
		"",
		"",
		"",
`f	22156dcd2c4932e15fcd679c1d127498afe06f85	C
f	3323a61ee353194428c344f1ee8baaa77cd78719	E
`, // /D
		"",
		"",
`d	dfb8f2ebf71da49c404b862cac704b84b98b7aed	D
f	0ab833f2f27597ce9a1d495f3427429ac0ef7dc0	A
`} // /B
	
	pathIndex := -1
	for i := 0; i < len(expectedPaths); i++ {
		sibling := -1
		rec := recs[i]
		switch {
		case rec.Block != nil:
			sibling = -2 // just a marker
		case rec.File != nil:
			pathIndex++
			sibling = int(rec.File.Sibling)
		case rec.Dir != nil:
			pathIndex++
			sibling = int(rec.Dir.Sibling)
			
			// Test directory content hashing
			if i < len(expectedContents) {
				expectStrong := StrongChecksum(
					bytes.NewBufferString(expectedContents[i]).Bytes())
				assert.Equalf(t, expectStrong, rec.Dir.Strong,
					"in path: %v", rec.Path)
			}
		}
		assert.Equal(t, expectedSiblings[i], sibling)
		if sibling != -2 {
			assert.Tf(t, strings.HasSuffix(rec.Path, expectedPaths[i]), 
				"%v lacks expected suffix %v", rec.Path, expectedPaths[i])
		}
	}
}

func TestRecIO(t *testing.T) {
	root := makeTree(t)
	defer os.RemoveAll(root)
	
	recout, err := ioutil.TempFile("", "recIO")
	assert.T(t, err == nil)
	defer os.Remove(recout.Name())
	
	walker := NewWalker(root)
	recChan := make(chan *ScanRec)
	walker.AddOutput(recChan)
	
	writer := NewRecWriter(recChan, recout)
	walker.Start()
	writer.Start()
	writer.Wait()
	
	recout.Sync()
	recout.Close()
	
	recin, err := os.Open(recout.Name())
	assert.T(t, err == nil)
	
	reader := NewRecReader(recin)
	readerChan := make(chan *ScanRec)
	reader.AddOutput(readerChan)
	
	walker = NewWalker(root)
	recChan = make(chan *ScanRec)
	walker.AddOutput(recChan)
	
	walker.Start()
	reader.Start()
	
	for {
		readRec, rok := <- readerChan
		walkRec, wok := <- recChan
		assert.Equal(t, rok, wok)
		if !wok || !rok { break }
		
		assert.Equalf(t, walkRec.Seq, readRec.Seq, "walk: %v read: %v", walkRec, readRec)
		switch {
		case walkRec.Block != nil:
			assert.T(t, readRec.Block != nil)
			assert.Equal(t, readRec.Block, walkRec.Block)
		case walkRec.File != nil:
			assert.T(t, readRec.File != nil)
			assert.Equal(t, readRec.File, walkRec.File)
		case walkRec.Dir != nil:
			assert.T(t, readRec.Dir != nil)
			assert.Equal(t, readRec.Dir, walkRec.Dir)
		default:
			assert.Tf(t, false, "Invalid record: %v", walkRec)
		}
	}
}

func TestCaskIndex(t *testing.T) {
	root := makeTree(t)
	defer os.RemoveAll(root)
	
	caskDir, err := ioutil.TempDir("", "caskIndex")
	assert.T(t, err == nil)
//	log.Printf("cask at: %s", caskDir)
	defer os.Remove(caskDir)
	
	cask, err := gocask.NewGocask(caskDir)
	assert.T(t, err == nil)
	defer cask.Close()
	assert.T(t, err == nil)
	
	walker := NewWalker(root)
	recChan := make(chan *ScanRec)
	walker.AddOutput(recChan)
	
	writer := NewCaskWriter(recChan, cask)
	
	walker.Start()
	writer.Start()
	writer.Wait()
	
	cask, err = gocask.NewGocask(caskDir)
	assert.T(t, err == nil)
	defer cask.Close()
	seqByStrong, err := cask.Get("dfb8f2ebf71da49c404b862cac704b84b98b7aed")
	assert.T(t, err == nil)
	assert.Equal(t, int32(4), bytesToInt(seqByStrong))
	
	nameBySeq, err := cask.Get(fmt.Sprintf("%x", 4))
	assert.T(t, err == nil)
	assert.Equal(t, "D", string(nameBySeq))
}
