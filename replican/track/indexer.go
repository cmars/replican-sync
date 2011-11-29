package track

import (
	"bytes"
	"gob"
	"log"
	"os"
	"path/filepath"

	"github.com/cmars/replican-sync/replican/fs"
)

// The Indexer maintains a hierarchical index of
// the directory structure. The indexer sends gobbed trees on any detected
// update in the filesystem.
type Indexer struct {
	Watcher Watcher
	Trees   chan []byte
	Filter  fs.IndexFilter
	tree    *fs.Dir
	exit    chan bool
	Log     *log.Logger
	errors  chan os.Error
}

func NewIndexer(watcher Watcher, log *log.Logger) *Indexer {
	if log == nil {
		log = NullLog()
	}
	indexer := &Indexer{
		Watcher: watcher,
		Trees:   make(chan []byte, 10),
		Filter:  excludeMetadata,
		exit:    make(chan bool, 1),
		Log:     log,
		errors:  logErrors(log)}
	go indexer.run()
	return indexer
}

func (indexer *Indexer) Stop() {
	indexer.exit <- true
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

func (indexer *Indexer) run() {
	indexer.tree = fs.IndexDir(indexer.Watcher.Root(), indexer.Filter, indexer.errors)
RUNNING:
	for {
		select {
		case paths := <-indexer.Watcher.Changes():
			indexer.Log.Printf("updating paths: %v", paths)
			indexer.UpdatePaths(paths)
			indexer.sendTree()
		case _ = <-indexer.exit:
			break RUNNING
		}
	}
	indexer.Log.Printf("exit")
	indexer.exit = nil
	close(indexer.errors)
	close(indexer.Trees)
}

func (indexer *Indexer) sendTree() os.Error {
	bufferEnc := bytes.NewBuffer([]byte{})
	encoder := gob.NewEncoder(bufferEnc)
	err := encoder.Encode(indexer.tree)
	if err == nil {
		indexer.Trees <- bufferEnc.Bytes()
	}
	return err
}

func (indexer *Indexer) UpdatePaths(paths []string) {
	for _, path := range paths {
		relpath := fs.MakeRelative(path, indexer.Watcher.Root())
		pathInfo, err := os.Stat(path)
		//		indexer.Log.Printf("update path %s: %v", relpath, pathInfo)
		if err != nil {
			indexer.Log.Printf("%s: does not exist (could be newly created)")
			continue
		}

		var pathIndex fs.FsNode
		if pathInfo.IsDirectory() {
			pathIndex = fs.IndexDir(path, indexer.Filter, indexer.errors)
		} else if pathInfo.IsRegular() {
			pathIndex, err = fs.IndexFile(path)
			if err == nil {
				indexer.Log.Printf("IndexFile %s: %v", path, err)
			}
		} else {
			// what is it?
			indexer.Log.Printf("%s: other type: %v", path, pathInfo)
			continue
		}

		if err == nil {
			indexer.updateNode(relpath, pathIndex)
		}
	}
	indexer.tree.UpdateStrong() // recalculate hashes all the way down
}

func (indexer *Indexer) updateNode(relpath string, newNode fs.FsNode) {
	if relpath == "" {
		indexer.Log.Printf("new root")
		indexer.tree = newNode.(*fs.Dir)
		return
	}

	oldNode, hasOld := indexer.tree.Resolve(relpath)

	parentPath, _ := filepath.Split(relpath)
	parentPath = filepath.Clean(parentPath)
	parentNode, hasParent := indexer.tree.Resolve(parentPath)
	if !hasParent {
		// wtf? parent does not resolve?
		indexer.Log.Printf("%s has no parent %s in tree? inconcievable!",
			relpath, parentPath)
		return
	}

	parentDir, isDir := parentNode.(*fs.Dir)
	if !isDir {
		// wat
		indexer.Log.Printf("parent %s %v: not a directory? inconcievable!",
			parentPath, parentNode)
		return
	}

	switch newNode.(type) {
	case *fs.Dir:
		indexer.Log.Printf("parent %s: append subdir %s", parentDir, relpath)
		parentDir.SubDirs = append(parentDir.SubDirs, newNode.(*fs.Dir))
	case *fs.File:
		indexer.Log.Printf("parent %s: append file %s", parentDir, relpath)
		parentDir.Files = append(parentDir.Files, newNode.(*fs.File))
	}

	if hasOld {
		indexer.Log.Printf("...replace")
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
