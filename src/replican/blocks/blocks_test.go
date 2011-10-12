
package blocks

import (
	"os"
	"replican/treegen"
	"strings"
	"testing"
	
	"github.com/bmizerany/assert"
)

func testIndexSomeMp3(t *testing.T) {
	var f *File
	var err os.Error
	
	cwd, _ := os.Getwd()
	t.Logf("CWD=%s", cwd)
	
	f, err = IndexFile("./testroot/My Music/0 10k 30.mp4")
	if f == nil {
		t.Fatalf("Failed to index file: %s", err.String())
	}
	
    assert.Equal(t, "5ab3e5d621402e5894429b5f595a1e2d7e1b3078", f.Strong())
    assert.Equal(t, "d1f11a93449fa4d3f320234743204ce157bbf1f3", f.Blocks[0].Strong())
    assert.Equal(t, "eabbe570b21cd2c5101a18b51a3174807fa5c0da", f.Blocks[1].Strong())
}

func testDirIndex(t *testing.T) {
	dir, _ := IndexDir("testroot/")
	
	assert.Equal(t, "10dc111ed3edd17ac89e303e877874aa61b45434", dir.Strong())
	
	var myMusic *Dir = dir.SubDirs[0]
	assert.Equal(t, "My Music", myMusic.Name())
	
	for i := 0; i < 2; i++ {
		var mp4file FsNode = myMusic.Files[i]
		assert.Tf(t, strings.HasPrefix(mp4file.Name(), "0 10k 30"),
			"Unexpected d -> d -> f name: %s", mp4file.Name())
	}
}

func testVisitDirsOnly(t *testing.T) {
	dir, _ := IndexDir("testroot/")
	collect := []*Dir{}
	visited := []Node{}
	
	Walk(dir, func(node Node) bool {
		visited = append(visited, node)
		
		d, ok := node.(*Dir)
		if ok {
			collect = append(collect, d)
			return true
		}
		
		_, ok = node.(*File)
		if ok {
			return false
		}
		
		t.Errorf("Unexpected type during visit: %v", node)
		return true
	})
	
	assert.Equalf(t, 3, len(collect), "Unexpected dirs in testroot/: %v", collect)
	
	for _, node := range visited {
		_, ok := node.(*Block)
		if ok {
			t.Fatalf("Should not have gotten a block, we told visitor to stop at file level.")
		}
	}
}

func testVisitBlocks(t *testing.T) {
	dir, _ := IndexDir("testroot/")
	collect := []*Block{}
	
	Walk(dir, func(node Node) bool {
		b, ok := node.(*Block)
		if ok {
			collect = append(collect, b)
		}
		
		return true
	})
	
	matched := false
	for _, block := range collect {
		if block.Strong() == "d1f11a93449fa4d3f320234743204ce157bbf1f3" {
			matched = true
		}
	}
	
	assert.Tf(t, matched, "Failed to find expected block")
}

func TestRelPath(t *testing.T) {
	tg := treegen.New()
	treeSpec := tg.D("foo", tg.F("bar", tg.B(42, 65537)))
	
	path := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(path)
	
	dir, err := IndexDir(path)
	assert.T(t, err == nil)
	
	assert.Equal(t, "", RelPath(dir))
	assert.Equal(t, "foo", RelPath(dir.SubDirs[0]))
	assert.Equal(t, "foo/bar", RelPath(dir.SubDirs[0].Files[0]))
	
	assert.Equal(t, "foo/bar", RelPath(dir.SubDirs[0].Files[0]))
}

