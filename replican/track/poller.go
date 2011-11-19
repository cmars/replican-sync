package track

import (
	"log"
	"os"
	"path/filepath"
	"time"
)

type Poller struct {
	Root      string
	Paths     chan string
	period_ns int64
	mtimes    map[string]int64
	changed   map[string]bool
	exit      chan bool
	timer     *time.Timer
	log       *log.Logger
}

func NewPoller(root string, period int) *Poller {
	poller := &Poller{
		Root:      filepath.Clean(root),
		Paths:     make(chan string, 20),
		period_ns: int64(period) * 1000000000,
		mtimes:    make(map[string]int64),
		exit:      make(chan bool, 2),
		log:       log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)}
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
		poller.log.Printf("changed: %s", path)
	}
}

func (poller *Poller) Poll() {
	poller.changed = make(map[string]bool)

	poller.log.Printf("begin scan")
	filepath.Walk(poller.Root, poller, nil)
	poller.log.Printf("scan complete")

	// Prune redundant entries.
	// Bounded by O(N*M) 
	// where N is total number of files & directories under root 
	// and M is max depth of the directory structure.
	sendPaths := []string{}
	for path, _ := range poller.changed {
		poller.log.Printf("check %s", path)

		pruned := false
		if path != poller.Root {
			parent, _ := filepath.Split(path)
			parent = filepath.Clean(parent)
			for parent != "" {
				poller.log.Printf("prune? %s vs %s", path, parent)
				if _, has := poller.changed[parent]; has {
					poller.log.Printf("prune! %s", path)
					pruned = true
					break
				}

				parent, _ = filepath.Split(parent)
				parent = filepath.Clean(parent)
			}
		}

		if !pruned {
			sendPaths = append(sendPaths, path)
		}
	}

	for _, path := range sendPaths {
		poller.log.Printf("send: %s", path)
		poller.Paths <- path
	}
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
	poller.exit = nil
	close(poller.Paths)
}
