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
	"github.com/kuroneko/gosqlite3"
)

func TestNop(t *testing.T) {
	dbpath, _ := ioutil.TempFile("", "test.db")
	dbpath.Close()
	defer os.RemoveAll(dbpath.Name())
	sqlite3.Session(dbpath.Name(), func(db *sqlite3.Database) {
	})
}

func TestCreateTable(t *testing.T) {
	dbpath, _ := ioutil.TempFile("", "test.db")
	dbpath.Close()
	defer os.RemoveAll(dbpath.Name())
	sqlite3.Session(dbpath.Name(), func(db *sqlite3.Database) {
		_, err := db.Execute("CREATE TABLE foo (name TEXT)")
		assert.T(t, err == nil)
		
		stmt, err := db.Prepare("INSERT INTO foo (name) VALUES (?)", "bar")
		stmt.Step()
		stmt.Finalize()
		
		stmt, err = db.Prepare("SELECT rowid, name FROM foo")
		stmt.Step()
		values := stmt.Row()
		assert.Equal(t, int64(1), values[0])
		assert.Equal(t, "bar", values[1])
		stmt.Finalize()
	})
}

func TestDbRepo(t *testing.T) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	dbpath, _ := ioutil.TempDir("", "test")
	defer os.RemoveAll(dbpath)
	
	dbrepo := NewDbRepo(dbpath)
	
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

