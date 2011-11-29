package track

import (
	"log"

	"github.com/cmars/replican-sync/replican/fs"
)

type Tracker struct {
	ckptLog *LocalCkptLog
	indexer *Indexer
	exit    chan bool
	log     *log.Logger
}

func NewTracker(indexer *Indexer, log *log.Logger) *Tracker {
	if log == nil {
		log = StdLog()
	}

	tracker := &Tracker{
		ckptLog: &LocalCkptLog{RootPath: indexer.Watcher.Root()},
		indexer: indexer,
		log:     log}

	tracker.ckptLog.Init()

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
		case treedata := <-tracker.indexer.Trees:
			tree, err := fs.DecodeDir(treedata)
			if err == nil {
				tracker.ckptLog.Append(tree)
			}
		case _ = <-tracker.exit:
			break RUNNING
		}
	}
}
