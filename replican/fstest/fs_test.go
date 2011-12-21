package fstest

import (
	"os"
	"github.com/cmars/replican-sync/replican/fs"
	"testing"

	"github.com/bmizerany/assert"
)

func TestFsIndexSomeMp3(t *testing.T) {
	cwd, _ := os.Getwd()
	t.Logf("CWD=%s", cwd)

	f, blks, err := fs.IndexFile("../../testroot/My Music/0 10k 30.mp4")
	if f == nil {
		t.Fatalf("Failed to index file: %v", err)
	}

	assert.Equal(t, "5ab3e5d621402e5894429b5f595a1e2d7e1b3078", f.Strong)
	assert.Equal(t, "d1f11a93449fa4d3f320234743204ce157bbf1f3", blks[0].Strong)
	assert.Equal(t, "eabbe570b21cd2c5101a18b51a3174807fa5c0da", blks[1].Strong)
}

func TestFsDirIndex(t *testing.T) {
	DoTestDirIndex(t, fs.NewMemRepo())
}

func TestFsVisitDirsOnly(t *testing.T) {
	DoTestVisitDirsOnly(t, fs.NewMemRepo())
}

func TestFsVisitBlocks(t *testing.T) {
	DoTestVisitBlocks(t, fs.NewMemRepo())
}

func TestFsNodeRelPath(t *testing.T) {
	DoTestNodeRelPath(t, fs.NewMemRepo())
}

func TestFsStoreRelPath(t *testing.T) {
	DoTestStoreRelPath(t, fs.NewMemRepo())
}

func TestFsDirResolve(t *testing.T) {
	DoTestDirResolve(t, fs.NewMemRepo())
}

func TestFsDirDescent(t *testing.T) {
	DoTestDirDescent(t, fs.NewMemRepo())
}

func TestFsParentRefs(t *testing.T) {
	DoTestParentRefs(t, fs.NewMemRepo())
}
