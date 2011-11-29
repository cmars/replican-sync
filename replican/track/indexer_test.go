package track

import (
	"bytes"
	"gob"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cmars/replican-sync/replican/fs"
	"github.com/cmars/replican-sync/replican/sync"
	"github.com/cmars/replican-sync/replican/treegen"

	"github.com/bmizerany/assert"
)

func TestindexerResponse(t *testing.T) {
	log := NullLog()
	tg := treegen.New()
	treeSpec := tg.D("root",
		tg.D("the jesus lizard",
			tg.D("goat",
				tg.F("then comes dudley", tg.B(78976, 655370)),
				tg.F("rodeo in joliet", tg.B(6712, 12891)),
				tg.F("south mouth", tg.B(1214, 123143)))),
		tg.D("rick astley",
			tg.D("greatest hits",
				tg.F("never gonna give you up"))),
		tg.D("sonic youth",
			tg.D("daydream nation",
				tg.F("teenage riot", tg.B(12903, 219301)),
				tg.F("cross the breeze", tg.B(13421, 219874)),
				tg.F("total trash", tg.B(70780, 90707)),
				tg.F("rain king", tg.B(2141221, 21321)))))
	path := treegen.TestTree(t, treeSpec)
	poller := NewPoller(filepath.Join(path, "root"), 1, NullLog())
	defer poller.Stop()
	indexer := NewIndexer(poller, NullLog())
	defer indexer.Stop()

	// When the indexer first starts up, it will get a scan of the entire root
	firstData := <-indexer.Trees
	firstRoot, err := fs.DecodeDir(firstData)
	log.Printf("Got initial root from indexer: %s", firstRoot.Strong())

	log.Printf("before patch1:\n")
	log.Printf("%v\n", fs.IndexDir(filepath.Join(path, "root"), indexer.Filter, nil))
	log.Printf("\n")

	// Append some bytes to a file, using replican sync.
	treeSpec = tg.D("root",
		tg.D("the jesus lizard",
			tg.D("goat",
				tg.F("then comes dudley", tg.B(78976, 655370), tg.B(9090, 100100)))))
	patchPath1 := treegen.TestTree(t, treeSpec)
	err = sync.Sync(patchPath1, path)
	assert.Tf(t, err == nil, "%v", err)

	patchTree1 := fs.IndexDir(filepath.Join(path, "root"), indexer.Filter, nil)

	// Give the poller 3s to find the change
	halt := time.After(3000000000)
	patchResults := []*fs.Dir{}
POLL1:
	for {
		select {
		case treedata := <-indexer.Trees:
			patchResult, err := fs.DecodeDir(treedata)
			assert.Tf(t, err == nil, "%v", err)
			patchResults = append(patchResults, patchResult)
		case _ = <-halt:
			break POLL1
		}
	}

	log.Printf("after patch1:\n")
	log.Printf("%v\n", patchTree1)
	log.Printf("\n")

	log.Printf("indexer reports from patch1:\n")
	log.Printf("%v\n", patchResults[0])
	log.Printf("\n")

	assert.Equal(t, 1, len(patchResults))
	assert.Equal(t, patchTree1.Strong(), patchResults[0].Strong())

	// Delete a file, see if scanner finds it
	rr := []string{path, "root", "rick astley", "greatest hits",
		"never gonna give you up"}
	err = os.Remove(filepath.Join(rr...))
	assert.T(t, err == nil)

	// Independent verification
	patchTree2 := fs.IndexDir(filepath.Join(path, "root"), indexer.Filter, nil)
	_, hasRR := patchTree2.Resolve(filepath.Join(rr[2:]...))
	assert.T(t, !hasRR)

	halt = time.After(3000000000)
	patchResults2 := []*fs.Dir{}
POLL2:
	for {
		select {
		case treedata := <-indexer.Trees:
			decoder := gob.NewDecoder(bytes.NewBuffer(treedata))
			patchResult := &fs.Dir{}
			err = decoder.Decode(patchResult)
			assert.Tf(t, err == nil, "%v", err)

			patchResults2 = append(patchResults2, patchResult)
		case _ = <-halt:
			break POLL2
		}
	}

	assert.Equal(t, 1, len(patchResults2))
	assert.Equal(t, patchTree2.Strong(), patchResults2[0].Strong())

	indexer.Stop()
}
