
package blocks

import (
	"os"
	"strings"
	"testing"
)

func TestIndexSomeMp3(t *testing.T) {
	var f *File
	var err os.Error
	
	cwd, _ := os.Getwd()
	t.Logf("CWD=%s", cwd)
	
	f, err = IndexFile("./testroot/My Music/0 10k 30.mp4")
	if f == nil {
		t.Fatalf("Failed to index file: %s", err.String())
	}
	
    if f.Strong() != "5ab3e5d621402e5894429b5f595a1e2d7e1b3078" {
    	t.Errorf("Unexpected strong file hash: %s", f.Strong())
    }
    t.Logf("file strong = %s", f.Strong())
    
    if f.Child(0).Strong() != "d1f11a93449fa4d3f320234743204ce157bbf1f3" {
    	t.Errorf("Unexpected block[0] hash: %s", f.Child(0).Strong())
    }
    
    if f.Child(1).Strong() != "eabbe570b21cd2c5101a18b51a3174807fa5c0da" {
    	t.Errorf("Unexpected block[1] hash: %s", f.Child(1).Strong())
    }
}

func TestDirIndex(t *testing.T) {
	dir, _ := IndexDir("testroot/")
	
	if dir.Strong() != "10dc111ed3edd17ac89e303e877874aa61b45434" {
		t.Errorf("Unexpected root directory hash: %s", dir.Strong())
	}
	
	var myMusic FsNode = dir.Child(0).(FsNode)
	if myMusic.Name() != "My Music" {
		t.Errorf("Expected My Music, got %s", myMusic.Name())
	}
	
	for i := 0; i < 2; i++ {
		var mp4file FsNode = myMusic.Child(i).(FsNode)
		if !strings.HasPrefix(mp4file.Name(), "0 10k 30") {
			t.Errorf("Unexpected d -> d -> f name: %s", mp4file.Name())
		}
	}
}

func TestVisitDirsOnly(t *testing.T) {
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
	
	if len(collect) != 3 {
		t.Errorf("Unexpected dirs in testroot/: %v", collect)
	}
	
	for _, node := range visited {
		_, ok := node.(*Block)
		if ok {
			t.Fatalf("Should not have gotten a block, we told visitor to stop at file level.")
		}
	}
}

func TestVisitBlocks(t *testing.T) {
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
	if !matched {
		t.Errorf("Failed to find expected block")
	}
}



