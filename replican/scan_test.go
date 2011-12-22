package replican

import (
	"log"
//	"os"
//	"strings"
	"testing"

	"github.com/cmars/replican-sync/replican/treegen"

//	"github.com/bmizerany/assert"
)

func TestSimpleScan(t *testing.T) {
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

SCANNING:
	for {
		select {
		case rec := <- scanner.Records:
			log.Printf("Got record: %v", rec)
		case path := <- scanner.Paths:
			log.Printf("Got path: %v", path)
		case <- scanner.Done:
			break SCANNING
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
