package track

import (
	"log"
	"path/filepath"
	"testing"
	"time"

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
	poller := NewPoller(filepath.Join(path, "root"), 1)
	log.Printf("Poller started.")

	halt := time.After(3000000000)

	paths := []string{}
TESTING:
	for {
		select {
		case path := <-poller.Paths:
			paths = append(paths, path)
		case _ = <-halt:
			poller.Stop()
			break TESTING
		}
	}

	assert.Equalf(t, 1, len(paths), "%v", paths)
	assert.Equalf(t, filepath.Join(path, "root"), paths[0], "%v", paths[0])
}
