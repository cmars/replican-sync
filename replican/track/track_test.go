package track

import (
	"os"
	"github.com/cmars/replican-sync/replican/treegen"
	"testing"

	"github.com/bmizerany/assert"
)

//func TestSomething(t *testing.T) {
//	assert.T(t, false)
//}

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
	
	log := &LocalCkptLog{ RootPath: path }
	err := log.Init()
	
	// Empty log, no head
	_, err = log.Head()
	assert.Tf(t, err == nil, "%v", err)
}
