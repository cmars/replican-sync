package replican

import (
	"bytes"
	"io/ioutil"
	"log"
//	"os"
	"strings"
	"testing"

	"github.com/cmars/replican-sync/replican/treegen"

	"github.com/bmizerany/assert"
)

func TestScanSiblings(t *testing.T) {
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
	
	scanner := NewScanner()
	scanner.Start(root)
	
	recs := []interface{}{}
	paths := []string{}
	
	l := log.New(ioutil.Discard, "", log.LstdFlags | log.Lshortfile)
//	l = log.New(os.Stdout, "", log.LstdFlags | log.Lshortfile)
	
	recNum := 0
SCANNING:
	for {
		select {
		case rec := <- scanner.Records:
			switch rec.(type) {
			case *BlockRec:
				blockRec := rec.(*BlockRec)
				l.Printf("%v %v", recNum, blockRec.Type)
			case *FileRec:
				fileRec := rec.(*FileRec)
				l.Printf("%v %v %x has sibling %v", recNum, fileRec.Type, fileRec.Strong, fileRec.Sibling)
			case *DirRec:
				dirRec := rec.(*DirRec)
				l.Printf("%v %v %x has sibling %v", recNum, dirRec.Type, dirRec.Strong, dirRec.Sibling)
			}
			recs = append(recs, rec)
			recNum++
		case path := <- scanner.Paths:
			paths = append(paths, path)
			l.Printf("Got path: %v", path)
		case <- scanner.Done:
			break SCANNING
		}
	}
	
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
		switch recs[i].(type) {
		case *BlockRec:
			sibling = -2 // just a marker
		case *FileRec:
			pathIndex++
			sibling = recs[i].(*FileRec).Sibling
		case *DirRec:
			pathIndex++
			sibling = recs[i].(*DirRec).Sibling
			
			// Test directory content hashing
			if i < len(expectedContents) {
				expectStrong := StrongChecksum(
					bytes.NewBufferString(expectedContents[i]).Bytes())
				assert.Equalf(t, expectStrong, recs[i].(*DirRec).Strong,
					"in path: %v", paths[pathIndex])
			}
		}
		assert.Equal(t, expectedSiblings[i], sibling)
		if sibling != -2 {
			assert.Tf(t, strings.HasSuffix(paths[pathIndex], expectedPaths[i]), 
				"%v lacks expected suffix %v", paths[pathIndex], expectedPaths[i])
		}
	}
}

/*
func TestPostOrder3(t *testing.T) {
	tg := treegen.New()
	treeSpec := tg.D("F",
		tg.D("B0",
			tg.D("A0",
				tg.F("C0", tg.B(100, 100)),
				tg.F("E0", tg.B(101, 100))),
			tg.F("D0", tg.B(102, 100))),
		tg.D("B1",
			tg.F("A", tg.B(1, 100)),
			tg.F("A2", tg.B(12, 100)),
			tg.D("D",
				tg.F("C", tg.B(2, 100)),
				tg.F("E", tg.B(3, 100)),
				tg.D("D2", tg.F("C2", tg.B(22, 100))))),
		tg.D("B2",
			tg.F("A3", tg.B(31, 100)),
			tg.D("D3",
				tg.F("C3", tg.B(32, 100)),
				tg.F("E3", tg.B(33, 100)))),
		tg.D("G",
			tg.D("I",
				tg.F("H", tg.B(4, 100)))))
	root := treegen.TestTree(t, treeSpec)
	order := []string{}
	PostOrderWalk(root, func(path string, info os.FileInfo, err error) error {
		order = append(order, path)
		return nil
	}, nil)

	/ *
		for _, s := range visitor.order {
			log.Printf("%v", s)
		}
	* /

	expect := []string{
		"/C0", "/E0", "/A0", "/D0", "/B0",
		"/C2", "/D2", "/C", "/E", "/D", "/A", "/A2", "/B1",
		"/C3", "/E3", "/D3", "/A3", "/B2",
		"/H", "/I", "/G",
		"/F"}
	for i := 0; i < len(expect); i++ {
		assert.Tf(t, strings.HasSuffix(order[i], expect[i]),
			"%v did not have expected suffix %s", order[i], expect[i])
	}
}
*/
