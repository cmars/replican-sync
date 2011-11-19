package track

import (
	//	"fmt"
	"github.com/cmars/replican-sync/replican/fs"
)

// The Tracker maintains a hierarchical index of
// the directory structure. The tree will be incrementally updated
// when the poller finds a change.
type Tracker struct {
	Poller *Poller
	Trees  chan *fs.Dir
	exit   chan bool
}

func NewTracker(poller *Poller) *Tracker {
	tracker := &Tracker{
		Poller: poller,
		Trees:  make(chan *fs.Dir),
		exit:   make(chan bool, 1)}
	go tracker.run()
	return tracker
}

func (tracker *Tracker) Stop() {
	tracker.exit <- true
}

func (tracker *Tracker) run() {
RUNNING:
	for {
		select {
		case _ = <-tracker.exit:
			break RUNNING
		}
	}
	tracker.exit = nil
	close(tracker.Trees)
}
