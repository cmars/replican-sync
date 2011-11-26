package track

import (
	"bytes"
	"gob"
	"log"
	"os"
	"path/filepath"

	"github.com/cmars/replican-sync/replican/fs"
)

// The Tracker maintains a hierarchical index of
// the directory structure. The tracker sends gobbed trees on any detected
// update in the filesystem.
type Tracker struct {
	Poller *Poller
	Trees  chan []byte
	Filter fs.IndexFilter
	tree   *fs.Dir
	exit   chan bool
	Log	   *log.Logger
	errors chan os.Error
}

func NewTracker(poller *Poller, log *log.Logger) *Tracker {
	if log == nil {
		log = NullLog()
	}
	tracker := &Tracker{
		Poller: poller,
		Trees:  make(chan []byte, 10),
		Filter: excludeMetadata,
		exit:   make(chan bool, 1),
		Log:    log,
		errors: logErrors(log) }
	go tracker.run()
	return tracker
}

func (tracker *Tracker) Stop() {
	tracker.exit <- true
}

func logErrors(log *log.Logger) chan os.Error {
	errors := make(chan os.Error, 1)
	go func() {
		for err := range errors {
			log.Printf("%v", err)
		}
	}()
	return errors
}

func (tracker *Tracker) run() {
	tracker.tree = fs.IndexDir(tracker.Poller.Root, tracker.Filter, tracker.errors)
RUNNING:
	for {
		select {
		case paths := <-tracker.Poller.Changed:
			tracker.Log.Printf("updating paths: %v", paths)
			tracker.UpdatePaths(paths)
			tracker.sendTree()
		case _ = <-tracker.exit:
			break RUNNING
		}
	}
	tracker.Log.Printf("exit")
	tracker.exit = nil
	close(tracker.errors)
	close(tracker.Trees)
}

func (tracker *Tracker) sendTree() os.Error {
	bufferEnc := bytes.NewBuffer([]byte{})
	encoder := gob.NewEncoder(bufferEnc)
	err := encoder.Encode(tracker.tree)
	if err == nil {
		tracker.Trees <- bufferEnc.Bytes()
	}
	return err
}

func (tracker *Tracker) UpdatePaths(paths []string) {
	for _, path := range paths {
		relpath := fs.MakeRelative(path, tracker.Poller.Root)
		pathInfo, err := os.Stat(path)
//		tracker.Log.Printf("update path %s: %v", relpath, pathInfo)
		if err != nil {
			tracker.Log.Printf("%s: does not exist (could be newly created)")
			continue
		}

		var pathIndex fs.FsNode
		if pathInfo.IsDirectory() {
			pathIndex = fs.IndexDir(path, tracker.Filter, tracker.errors)
		} else if pathInfo.IsRegular() {
			pathIndex, err = fs.IndexFile(path)
			if err == nil {
				tracker.Log.Printf("IndexFile %s: %v", path, err)
			}
		} else {
			// what is it?
			tracker.Log.Printf("%s: other type: %v", path, pathInfo)
			continue
		}

		if err == nil {
			tracker.updateNode(relpath, pathIndex)
		}
	}
	tracker.tree.UpdateStrong() // recalculate hashes all the way down
}

func (tracker *Tracker) updateNode(relpath string, newNode fs.FsNode) {
	if relpath == "" {
		tracker.Log.Printf("new root")
		tracker.tree = newNode.(*fs.Dir)
		return
	}

	oldNode, hasOld := tracker.tree.Resolve(relpath)

	parentPath, _ := filepath.Split(relpath)
	parentPath = filepath.Clean(parentPath)
	parentNode, hasParent := tracker.tree.Resolve(parentPath)
	if !hasParent {
		// wtf? parent does not resolve?
		tracker.Log.Printf("%s has no parent %s in tree? inconcievable!", 
			relpath, parentPath)
		return
	}

	parentDir, isDir := parentNode.(*fs.Dir)
	if !isDir {
		// wat
		tracker.Log.Printf("parent %s %v: not a directory? inconcievable!", 
			parentPath, parentNode)
		return
	}

	switch newNode.(type) {
	case *fs.Dir:
		tracker.Log.Printf("parent %s: append subdir %s", parentDir, relpath) 
		parentDir.SubDirs = append(parentDir.SubDirs, newNode.(*fs.Dir))
	case *fs.File:
		tracker.Log.Printf("parent %s: append file %s", parentDir, relpath) 
		parentDir.Files = append(parentDir.Files, newNode.(*fs.File))
	}

	if hasOld {
		tracker.Log.Printf("...replace") 
		switch oldNode.(type) {
		case *fs.Dir:
			parentDir.SubDirs = removeDir(parentDir.SubDirs, oldNode.(*fs.Dir))
		case *fs.File:
			parentDir.Files = removeFile(parentDir.Files, oldNode.(*fs.File))
		}
	}
}

func removeDir(slice []*fs.Dir, item *fs.Dir) []*fs.Dir {
	result := []*fs.Dir{}
	for _, elem := range slice {
		if item != elem {
			result = append(result, elem)
		}
	}
	return result
}

func removeFile(slice []*fs.File, item *fs.File) []*fs.File {
	result := []*fs.File{}
	for _, elem := range slice {
		if item != elem {
			result = append(result, elem)
		}
	}
	return result
}
