package track

import (
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"
)

// Periodically polls a directory tree for changes in 
// file and directory modified times.
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

func NewPoller(root string, period int, writer io.Writer) *Poller {
	if writer == nil {
		writer = ioutil.Discard
	}

	poller := &Poller{
		Root:      filepath.Clean(root),
		Paths:     make(chan string, 20),
		period_ns: int64(period) * 1000000000,
		mtimes:    make(map[string]int64),
		exit:      make(chan bool, 1),
		log:       log.New(writer, "", log.LstdFlags|log.Lshortfile)}
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

		if !poller.hasChangedParent(path) {
			sendPaths = append(sendPaths, path)
		}
	}

	for _, path := range sendPaths {
		poller.log.Printf("send: %s", path)
		poller.Paths <- path
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
	poller.exit = nil
	close(poller.Paths)
}
