package fstest

import (
	"os"
	"github.com/cmars/replican-sync/replican/fs/sqlite3"
	"io/ioutil"
	"testing"
)

func TestDbDirIndex(t *testing.T) {
	dbpath, _ := ioutil.TempDir("", "dbtest")
	defer os.RemoveAll(dbpath)
	DoTestDirIndex(t, sqlite3.NewDbRepo(dbpath))
}

func TestDbVisitDirsOnly(t *testing.T) {
	dbpath, _ := ioutil.TempDir("", "dbtest")
	defer os.RemoveAll(dbpath)
	DoTestVisitDirsOnly(t, sqlite3.NewDbRepo(dbpath))
}

func TestDbVisitBlocks(t *testing.T) {
	dbpath, _ := ioutil.TempDir("", "dbtest")
	defer os.RemoveAll(dbpath)
	DoTestVisitBlocks(t, sqlite3.NewDbRepo(dbpath))
}

func TestDbNodeRelPath(t *testing.T) {
	dbpath, _ := ioutil.TempDir("", "dbtest")
	defer os.RemoveAll(dbpath)
	DoTestNodeRelPath(t, sqlite3.NewDbRepo(dbpath))
}

func TestDbStoreRelPath(t *testing.T) {
	dbpath, _ := ioutil.TempDir("", "dbtest")
	defer os.RemoveAll(dbpath)
	DoTestStoreRelPath(t, sqlite3.NewDbRepo(dbpath))
}

func TestDbDirResolve(t *testing.T) {
	dbpath, _ := ioutil.TempDir("", "dbtest")
	defer os.RemoveAll(dbpath)
	DoTestDirResolve(t, sqlite3.NewDbRepo(dbpath))
}

func TestDbDirDescent(t *testing.T) {
	dbpath, _ := ioutil.TempDir("", "dbtest")
	defer os.RemoveAll(dbpath)
	DoTestDirDescent(t, sqlite3.NewDbRepo(dbpath))
}

func TestDbParentRefs(t *testing.T) {
	dbpath, _ := ioutil.TempDir("", "dbtest")
	defer os.RemoveAll(dbpath)
	DoTestParentRefs(t, sqlite3.NewDbRepo(dbpath))
}
