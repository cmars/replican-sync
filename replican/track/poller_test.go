package track

import (
	//	"fmt"
	//	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cmars/replican-sync/replican/sync"
	"github.com/cmars/replican-sync/replican/treegen"

	"github.com/bmizerany/assert"
)

func TestPollerInit(t *testing.T) {
	tg := treegen.New()
	treeSpec := tg.D("root",
		tg.D("the jesus lizard",
			tg.D("goat",
				tg.F("then comes dudley", tg.B(78976, 655370)),
				tg.F("rodeo in joliet", tg.B(6712, 12891)),
				tg.F("south mouth", tg.B(1214, 123143)))),
		tg.D("sonic youth",
			tg.D("daydream nation",
				tg.F("teenage riot", tg.B(12903, 219301)),
				tg.F("cross the breeze", tg.B(13421, 219874)),
				tg.F("rain king", tg.B(2141221, 21321)))))
	path := treegen.TestTree(t, treeSpec)
	poller := NewPoller(filepath.Join(path, "root"), 1, nil /*os.Stdout*/ )

	halt := time.After(3000000000)

	respPaths := []string{}
TESTING:
	for {
		select {
		case paths := <-poller.Changed:
			respPaths = append(respPaths, paths...)
		case _ = <-halt:
			poller.Stop()
			break TESTING
		}
	}

	assert.Equalf(t, 1, len(respPaths), "%v", respPaths)
	assert.Equal(t, filepath.Join(path, "root"), respPaths[0])
}

func TestPollerResponse(t *testing.T) {
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

	expectRoot := <-poller.Changed
	assert.Equal(t, filepath.Join(path, "root"), expectRoot[0])

	// Append some bytes to a file, using replican sync.
	treeSpec = tg.D("root",
		tg.D("the jesus lizard",
			tg.D("goat",
				tg.F("then comes dudley", tg.B(78976, 655370), tg.B(9090, 100100)))))
	patchPath1 := treegen.TestTree(t, treeSpec)
	err := sync.Sync(patchPath1, path)
	assert.Tf(t, err == nil, "%v", err)

	// Give the poller 3s to find the change
	halt := time.After(3000000000)
	patchResult1 := []string{}
POLL1:
	for {
		select {
		case paths := <-poller.Changed:
			patchResult1 = append(patchResult1, paths...)
		case _ = <-halt:
			break POLL1
		}
	}

	assert.Equal(t, 1, len(patchResult1))
	assert.Equal(t,
		filepath.Join(path, "root", "the jesus lizard", "goat"),
		patchResult1[0])

	// Delete a file, see if scanner finds it
	rr := []string{path, "root", "rick astley", "greatest hits",
		"never gonna give you up"}
	err = os.Remove(filepath.Join(rr...))
	assert.T(t, err == nil)

	halt = time.After(3000000000)
	patchResult2 := []string{}
POLL2:
	for {
		select {
		case paths := <-poller.Changed:
			patchResult2 = append(patchResult2, paths...)
		case _ = <-halt:
			break POLL2
		}
	}

	assert.Equal(t, 1, len(patchResult2))
	assert.Equal(t, filepath.Join(rr[:4]...), patchResult2[0])

	poller.Stop()
}
