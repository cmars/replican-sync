package track

import (
	"os"
	"path/filepath"
	"time"
)

type Poller struct {
	Root string
	Paths chan string
	period_ns int64
	mtimes map[string]int64
	changed map[string]bool
	exit chan bool
	timer *time.Timer
}

func NewPoller(root string, period int) *Poller {
	poller := &Poller {
		Root: root,
		Paths: make(chan string),
		period_ns: int64(period)*1000000000,
		mtimes: make(map[string]int64),
		exit: make(chan bool) }
	go poller.run()
	return poller
}

func (poller *Poller) Stop() {
	poller.exit <- true
}

func (poller *Poller) VisitDir(path string, f *os.FileInfo) bool {
	poller.visit(path, f)
	return true
}

func (poller *Poller) VisitFile(path string, f *os.FileInfo) {
	poller.visit(path, f)
}

func (poller *Poller) visit(path string, f *os.FileInfo) {
	curMtime, has := poller.mtimes[path]
	if !has || f.Mtime_ns > curMtime {
		poller.mtimes[path] = f.Mtime_ns
		poller.changed[path] = true
	}
}

func (poller *Poller) Poll() {
	poller.changed = make(map[string]bool)
	filepath.Walk(poller.Root, poller, nil)
	
	// Prune redundant entries
	// (I know this is a sucky way to go about it...)
	for path, _ := range poller.changed {
		for parent, _ := filepath.Split(path);
				parent != poller.Root && parent != "";
				parent, _ = filepath.Split(parent) {
			if _, has := poller.changed[parent]; has {
				poller.changed[path] = false, false
			}
		}
	}
	
	for path, _ := range poller.changed {
		poller.Paths <- path
	}
}

func (poller *Poller) run() {
	poller.timer = time.NewTimer(poller.period_ns)
	for {
		select {
		case _ = <-poller.timer.C:
			poller.Poll()
			poller.timer = time.NewTimer(poller.period_ns)
		case _ = <-poller.exit:
			return
		}
	}
}
