package track

import (
	"log"
	"os"
	"path/filepath"
	"time"
)

// Periodically polls a directory tree for changes in 
// file and directory modified times.
type Poller struct {
	Root      string
	Changed   chan []string
	period_ns int64
	mtimes    map[string]int64
	changed   map[string]bool
	exit      chan bool
	timer     *time.Timer
	Log       *log.Logger
}

func NewPoller(root string, period int, log *log.Logger) *Poller {
	if log == nil {
		log = NullLog()
	}
	poller := &Poller{
		Root:      filepath.Clean(root),
		Changed:   make(chan []string, 20),
		period_ns: int64(period) * 1000000000,
		mtimes:    make(map[string]int64),
		exit:      make(chan bool, 1),
		Log:       log}
	go poller.run()
	return poller
}

func (poller *Poller) Stop() {
	if poller.exit != nil {
		poller.exit <- true
	}
}

func (poller *Poller) VisitDir(path string, f *os.FileInfo) bool {
	poller.visit(path, f)
	return true
}

func (poller *Poller) VisitFile(path string, f *os.FileInfo) {
	poller.visit(path, f)
}

func (poller *Poller) visit(path string, f *os.FileInfo) {
	path = filepath.Clean(path)
	curMtime, has := poller.mtimes[path]
	if !has || f.Mtime_ns > curMtime {
		poller.mtimes[path] = f.Mtime_ns
		poller.changed[path] = true
		poller.Log.Printf("changed: %s", path)
	}
}

func (poller *Poller) Poll() {
	poller.changed = make(map[string]bool)

	poller.Log.Printf("begin scan")
	filepath.Walk(poller.Root, poller, nil)
	poller.Log.Printf("scan complete")

	// Prune redundant entries.
	// Bounded by O(N*M) 
	// where N is total number of files & directories under root 
	// and M is max depth of the directory structure.
	// (best case: NlogN, worst case: N^2)
	sendPaths := []string{}
	for path, _ := range poller.changed {
		poller.Log.Printf("check %s", path)

		if !poller.hasChangedParent(path) {
			sendPaths = append(sendPaths, path)
		}
	}
	
	if len(sendPaths) > 0 {
		poller.Log.Printf("send: %s", sendPaths)
		poller.Changed <- sendPaths
	}
}

func (poller *Poller) hasChangedParent(path string) bool {
	for parent, _ := filepath.Split(path); parent != "/"; parent, _ = filepath.Split(parent) {
		parent = filepath.Clean(parent)
		if _, has := poller.changed[parent]; has {
			return true
		}
	}
	return false
}

func (poller *Poller) run() {
	poller.timer = time.NewTimer(poller.period_ns)
RUNNING:
	for {
		select {
		case _ = <-poller.timer.C:
			poller.Poll()
			poller.timer = time.NewTimer(poller.period_ns)
		case _ = <-poller.exit:
			break RUNNING
		}
	}
	poller.Log.Printf("exit")
	poller.exit = nil
	close(poller.Changed)
}
