
package blocks

import (
	"os"
	"testing"
)

func TestIndexSomeMp3(t *testing.T) {
	var f *File
	var err os.Error
	
	f, err = IndexFile("./testroot/My Music/0 10k 30.mp4")
	if f == nil {
		t.Fatalf("Failed to index file: %s", err.String())
	}
	
    if f.Strong != "5ab3e5d621402e5894429b5f595a1e2d7e1b3078" {
    	t.Errorf("Unexpected strong file hash: %s", f.Strong)
    }
    t.Logf("file strong = %s", f.Strong)
    
    if f.Blocks[0].Strong != "d1f11a93449fa4d3f320234743204ce157bbf1f3" {
    	t.Errorf("Unexpected block[0] hash: %s", f.Blocks[0].Strong)
    }
    
    if f.Blocks[1].Strong != "eabbe570b21cd2c5101a18b51a3174807fa5c0da" {
    	t.Errorf("Unexpected block[1] hash: %s", f.Blocks[0].Strong)
    }
}


