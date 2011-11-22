package track

import (
	"bytes"
	"gob"
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
}

func NewTracker(poller *Poller) *Tracker {
	tracker := &Tracker{
		Poller: poller,
		Trees:  make(chan []byte, 10),
		Filter: excludeMetadata,
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
		case paths := <-tracker.Poller.Changed:
			tracker.UpdatePaths(paths)
			tracker.sendTree()
		case _ = <-tracker.exit:
			break RUNNING
		}
	}
	tracker.exit = nil
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
		if err != nil {
			continue
		}
		
		var pathIndex fs.FsNode
		if pathInfo.IsDirectory() {
			pathIndex = fs.IndexDir(path, tracker.Filter, nil)
		} else if pathInfo.IsRegular() {
			pathIndex, err = fs.IndexFile(path)
		} else {
			// what is it?
			continue
		}
		
		if err != nil {
			tracker.tree = updateNode(tracker.tree, relpath, pathIndex)
		}
	}
	tracker.tree.Strong() // recalculate hashes all the way down
}

func updateNode(
		tree *fs.Dir, relpath string, newNode fs.FsNode) *fs.Dir {
	if relpath == "" {
		return newNode.(*fs.Dir)
	}
	
	oldNode, hasOld := tree.Resolve(relpath)
	
	parentPath, _ := filepath.Split(relpath)
	parentNode, hasParent := tree.Resolve(parentPath)
	if !hasParent {
		// wtf? parent does not resolve?
		return tree
	}
	
	parentDir, isDir := parentNode.(*fs.Dir)
	if !isDir {
		// wat
		return tree
	}
	
	switch newNode.(type) {
	case *fs.Dir:
		parentDir.SubDirs = append(parentDir.SubDirs, newNode.(*fs.Dir))
	case *fs.File:
		parentDir.Files = append(parentDir.Files, newNode.(*fs.File))
	}
	
	if hasOld {
		switch oldNode.(type) {
		case *fs.Dir:
			parentDir.SubDirs = removeDir(parentDir.SubDirs, oldNode.(*fs.Dir))
		case *fs.File:
			parentDir.Files = removeFile(parentDir.Files, oldNode.(*fs.File))
		}
	}
	return tree
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
