package track

import (
	"time"
	"github.com/cmars/replican-sync/replican/fs"
)

type ScannerUpdate struct {
	Root *fs.Dir
}

// Start a periodic scan of a local directory store in a 
// new goroutine. Every 'delay' seconds, the directory will 
// be reindexed and checked for changes. If a change is found, 
// the new change is sent to updateChan. Any value sent to 
// endChan will terminate the scanner goroutine.
func StartPeriodicScan(store *fs.LocalDirStore, delay int) (<-chan *ScannerUpdate, chan<- bool) {
	updateChan := make(chan *ScannerUpdate)
	endChan := make(chan bool)

	go func() {
		for _, end := <-endChan; !end; _, end = <-endChan {
			update := &ScannerUpdate{}
			update.Root = fs.IndexDir(store.RootPath(), nil)
			if update.Root != nil && (update.Root.Strong() != store.Root().Strong()) {
				updateChan <- update
			}

			time.Sleep(int64(delay) * int64(1000000000))
		}
		close(updateChan)
	}()
	return updateChan, endChan
}
