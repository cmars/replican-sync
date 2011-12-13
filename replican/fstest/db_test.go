package fstest

import (
	"os"
	"github.com/cmars/replican-sync/replican/fs/sqlite3"
	"io/ioutil"
	"testing"
	
	"github.com/bmizerany/assert"
)

func createDbRepo(t *testing.T) (*sqlite3.DbRepo, string) {
	dbpath, _ := ioutil.TempFile("", "test.db")
	dbpath.Close()
	dbrepo, err := sqlite3.NewDbRepo(dbpath.Name())
	assert.T(t, err == nil)
	return dbrepo, dbpath.Name()
}

func TestDbDirIndex(t *testing.T) {
	dbrepo, dbpath := createDbRepo(t)
	defer os.RemoveAll(dbpath)
	DoTestDirIndex(t, dbrepo)
}

func TestDbVisitDirsOnly(t *testing.T) {
	dbrepo, dbpath := createDbRepo(t)
	defer os.RemoveAll(dbpath)
	DoTestVisitDirsOnly(t, dbrepo)
}

func TestDbVisitBlocks(t *testing.T) {
	dbrepo, dbpath := createDbRepo(t)
	defer os.RemoveAll(dbpath)
	DoTestVisitBlocks(t, dbrepo)
}

func TestDbNodeRelPath(t *testing.T) {
	dbrepo, dbpath := createDbRepo(t)
	defer os.RemoveAll(dbpath)
	DoTestNodeRelPath(t, dbrepo)
}

func TestDbStoreRelPath(t *testing.T) {
	dbrepo, dbpath := createDbRepo(t)
	defer os.RemoveAll(dbpath)
	DoTestStoreRelPath(t, dbrepo)
}

func TestDbDirResolve(t *testing.T) {
	dbrepo, dbpath := createDbRepo(t)
	defer os.RemoveAll(dbpath)
	DoTestDirResolve(t, dbrepo)
}

func TestDbDirDescent(t *testing.T) {
	dbrepo, dbpath := createDbRepo(t)
	defer os.RemoveAll(dbpath)
	DoTestDirDescent(t, dbrepo)
}

func TestDbParentRefs(t *testing.T) {
	dbrepo, dbpath := createDbRepo(t)
	defer os.RemoveAll(dbpath)
	DoTestParentRefs(t, dbrepo)
}
