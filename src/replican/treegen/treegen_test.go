
package treegen

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	
	"github.com/bmizerany/assert"
)

func TestSimple1(t *testing.T) {
	tg := New()
	treeSpec := tg.D("foo", tg.F("bar", tg.B(42, 65537)))
	
	err = FabTree(t, treeSpec)
	if err != nil { fmt.Printf(err.String()) }
	assert.Tf(t, err == nil, "Failed to create tree")
	
	fileInfo, err := os.Stat(filepath.Join(tempdir, "foo"))
	assert.Tf(t, fileInfo.IsDirectory(), "no foo")
	
	fileInfo, err = os.Stat(filepath.Join(tempdir, "foo", "bar"))
	assert.Tf(t, fileInfo.IsRegular(), "no bar")
	assert.Equal(t, int64(65537), fileInfo.Size)
}



