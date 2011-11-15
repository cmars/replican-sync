package track

import (
	"os"
	//	"fmt"
	"path/filepath"
	"testing"

	"github.com/cmars/replican-sync/replican/sync"
	"github.com/cmars/replican-sync/replican/treegen"

	"github.com/bmizerany/assert"
)

func TestCheckpointLog(t *testing.T) {
	tg := treegen.New()
	treeSpec := tg.D("root",
		tg.D("ladytron",
			tg.D("604",
				tg.F("discotraxx", tg.B(78976, 655370)),
				tg.F("commodore rock", tg.B(1214, 123143))),
			tg.D("light and magic",
				tg.F("true mathematics", tg.B(1, 12121), tg.B(2, 23232)),
				tg.F("cracked lcd", tg.B(3, 34343), tg.B(4, 45454)))),
		tg.D("bedrock",
			tg.F("heaven scent", tg.B(909, 10101))))

	path := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(path)
	//	fmt.Printf("%s\n", path)

	log := &LocalCkptLog{RootPath: path}
	err := log.Init()

	// Empty log, no head
	ckpt0, err := log.Head()
	assert.T(t, err == nil)
	assert.Equal(t, int64(0), ckpt0.Tstamp())
	assert.Equal(t, "", ckpt0.Strong())

	// Create the first checkpoint
	ckpt1, err := log.Snapshot()
	assert.Tf(t, err == nil, "%v", err)

	// Head should now be non-empty, local checkpoint
	head1, err := log.Head()
	assert.Tf(t, err == nil, "%v", err)
	assert.Equal(t, ckpt1.Strong(), head1.Strong())

	assert.Tf(t, ckpt1.Tstamp() > int64(0), "unexpected checkpoint tstamp: %d", ckpt1.Tstamp())
	assert.Equal(t, len(ckpt1.Parents()), 0)
	assert.Tf(t, ckpt1.Strong() != "", "unexpected checkpoint strong: '%v'", ckpt1.Strong())

	// Oops, forgot some songs!
	// Let's sync some over.
	tg = treegen.New()
	treeSpec = tg.D("root",
		tg.D("ladytron",
			tg.D("604",
				tg.F("playgirl", tg.B(90123, 10293))),
			tg.D("light and magic",
				tg.F("blue jeans", tg.B(123, 45678)))),
		tg.D("bedrock",
			tg.F("beautiful strange", tg.B(909, 10101))))
	appendPath := treegen.TestTree(t, treeSpec)
	err = sync.Sync(filepath.Join(appendPath, "root"), filepath.Join(path, "root"))
	assert.Tf(t, err == nil, "%v", err)

	// Create another checkpoint
	ckpt2, err := log.Snapshot()
	assert.Tf(t, err == nil, "%v", err)
	head2, err := log.Head()
	assert.Tf(t, err == nil, "%v", err)
	assert.Equal(t, ckpt2.Strong(), head2.Strong())
	assert.Tf(t, ckpt1.Strong() != ckpt2.Strong(), "checkpoints should have changed!")
}
