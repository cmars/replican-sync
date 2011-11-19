package track

import (
	"log"
	"os"
	"os/inotify"
	"path/filepath"
	"time"
)

const watchMask = (inotify.IN_CREATE | inotify.IN_DELETE |
	inotify.IN_DELETE_SELF | inotify.IN_MODIFY |
	inotify.IN_MOVE | inotify.IN_MOVE_SELF)

type Watcher struct {
	*inotify.Watcher
	Root          string
	updateChan    chan string
	settlingChans map[string]chan *inotify.Event
	exit          chan bool
}

func (watcher *Watcher) VisitDir(path string, f *os.FileInfo) bool {
	watcher.AddWatch(path, inotify.IN_ONLYDIR)
	return true
}

func (watcher *Watcher) VisitFile(path string, f *os.FileInfo) {

}

func NewWatcher(root string, period int) (*Watcher, os.Error) {
	inotifyWatcher, err := inotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	watcher := &Watcher{
		Watcher:       inotifyWatcher,
		Root:          root,
		updateChan:    make(chan string),
		settlingChans: make(map[string]chan *inotify.Event),
		exit:          make(chan bool)}

	filepath.Walk(root, watcher, nil)

	go watcher.run()

	return watcher, nil
}

func (watcher *Watcher) run() {
	for {
		select {
		case ev := <-watcher.Event:
			if ev.Mask&(inotify.IN_CREATE|inotify.IN_ISDIR) != 0 {
				watcher.AddWatch(ev.Name, inotify.IN_ONLYDIR)
			}

			settleChan, isSettling := watcher.settlingChans[ev.Name]
			if !isSettling {
				settleChan = watcher.newSettlePath(ev.Name)
			}
			settleChan <- ev

		case err := <-watcher.Error:
			log.Println("Error watching %s: %v", watcher.Root, err)

		case _ = <-watcher.exit:
			return
		}
	}
}

// Create a goroutine that settles a path.
// It waits for a given path to stop recieving updates for 
// a given 
func (watcher *Watcher) newSettlePath(path string) chan *inotify.Event {
	incoming := make(chan *inotify.Event)
	timer := time.NewTimer(15 * 1000000000)
	go func() {
		for {
			select {
			case _ = <-incoming:
				// Reset the timer
				if timer.Stop() {
					timer = time.NewTimer(15 * 1000000000)
				} else {
					return
				}
			case _ = <-timer.C:
				watcher.updateChan <- path
				return
			}
		}
	}()

	return incoming
}
