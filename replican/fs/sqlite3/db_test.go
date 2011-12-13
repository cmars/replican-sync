package sqlite3	

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"
	
	"github.com/cmars/replican-sync/replican/fs"
	"github.com/cmars/replican-sync/replican/treegen"
	
	"github.com/bmizerany/assert"
)

func createDbRepo(t *testing.T) (*DbRepo, string) {
	dbpath, _ := ioutil.TempFile("", "test.db")
	dbpath.Close()
	dbrepo, err := NewDbRepo(dbpath.Name())
	assert.T(t, err == nil)
	return dbrepo, dbpath.Name()
}

func TestDbRepo(t *testing.T) {
	dbrepo, dbpath := createDbRepo(t)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	defer os.Remove(dbpath)
	defer dbrepo.Close()
	
	tg := treegen.New()
	treeSpec := tg.D("foo",
		tg.D("bar",
			tg.D("aleph",
				tg.F("A", tg.B(42, 65537)),
				tg.F("a", tg.B(42, 65537))),
			tg.D("beth",
				tg.F("B", tg.B(43, 65537)),
				tg.F("b", tg.B(43, 65537))),
			tg.D("jimmy",
				tg.F("G", tg.B(44, 65537)),
				tg.F("g", tg.B(44, 65537)))),
		tg.D("baz",
			tg.D("uno",
				tg.F("1", tg.B(1, 65537)),
				tg.F("I", tg.B(1, 65537))),
			tg.D("dos",
				tg.F("2", tg.B(11, 65537)),
				tg.F("II", tg.B(11, 65537))),
			tg.D("tres",
				tg.F("3", tg.B(111, 65537)),
				tg.F("III", tg.B(111, 65537)))))

	path := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(path)
	
	foo := fs.IndexDir(filepath.Join(path, "foo"), dbrepo, nil)
	
	fmt.Printf("%v\n", foo.Info())
}
