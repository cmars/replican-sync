package fs

import (
//	"log"
	"os"
	"strings"
	"testing"
	
	"github.com/cmars/replican-sync/replican/treegen"
	
	"github.com/bmizerany/assert"
)

type postVisitor struct {
	order []string
}

func (pv *postVisitor) VisitDir(path string, f *os.FileInfo) bool {
	pv.order = append(pv.order, path)
	return true
}

func (pv *postVisitor) VisitFile(path string, f *os.FileInfo) {
	pv.order = append(pv.order, path)
}

func TestPostOrder(t *testing.T) {
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
	visitor := &postVisitor{ order: []string{} }
	PostOrderWalk(root, visitor, nil)
	
	/*
	for _, s := range visitor.order {
		log.Printf("%v", s)
	}
	*/
	
	expect := []string{ "/C", "/E", "/D", "/A", "/B", "/H", "/I", "/G", "/F" }
	for i := 0; i < len(expect); i++ {
		assert.Tf(t, strings.HasSuffix(visitor.order[i], expect[i]), 
			"%v did not have expected suffix %s", visitor.order[i], expect[i])
	}
}

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
	visitor := &postVisitor{ order: []string{} }
	PostOrderWalk(root, visitor, nil)
	
	/*
	for _, s := range visitor.order {
		log.Printf("%v", s)
	}
	*/
	
	expect := []string{ 
		"/C0", "/E0", "/A0", "/D0", "/B0",
		"/C2", "/D2", "/C", "/E", "/D", "/A", "/A2", "/B1",
		"/C3", "/E3", "/D3", "/A3", "/B2", 
		"/H", "/I", "/G", 
		"/F" }
	for i := 0; i < len(expect); i++ {
		assert.Tf(t, strings.HasSuffix(visitor.order[i], expect[i]), 
			"%v did not have expected suffix %s", visitor.order[i], expect[i])
	}
}


