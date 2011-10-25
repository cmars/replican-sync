
package treegen

import (
	"os"
	"path/filepath"
	"testing"
	
	"github.com/bmizerany/assert"
)

func TestSimple1(t *testing.T) {
	tg := New()
	treeSpec := tg.D("foo", tg.F("bar", tg.B(42, 65537)))
	
	tempdir := TestTree(t, treeSpec)
	defer os.RemoveAll(tempdir)
	
	fileInfo, _ := os.Stat(filepath.Join(tempdir, "foo"))
	assert.Tf(t, fileInfo.IsDirectory(), "no foo")
	
	fileInfo, _ = os.Stat(filepath.Join(tempdir, "foo", "bar"))
	assert.Tf(t, fileInfo.IsRegular(), "no bar")
	assert.Equal(t, int64(65537), fileInfo.Size)
}



