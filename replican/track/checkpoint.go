package track

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"fmt"
	"gob"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/cmars/replican-sync/replican/fs"
)

const METADATA_NAME = ".replican"

// A Checkpoint represents the state of a directory at a point in time.
type Checkpoint interface {

	// Representation of the tree and all its contents.
	Root() *fs.Dir

	// Parent checkpoints which preceded this state of the tree.
	// Every checkpoint, except the first, will have at least one parent.
	// A checkpoint that results from a merge operation will have multiple parents.
	Parents() []Checkpoint

	// Strong checksum of the checkpoint.
	// This is a function of the tree checksum, timestamp, and lineage.
	Strong() string

	// When the checkpoint was taken.
	Tstamp() int64
}

// A Checkpoint log tracks changes in a directory over time.
type CheckpointLog interface {

	// Fetch the checkpoint with given strong checksum
	Checkpoint(strong string) (Checkpoint, os.Error)

	// Fetch the current head checkpoint.
	Head() (Checkpoint, os.Error)

	// Create a checkpoint of the current block store state 
	// and append to the head of the log. 
	Snapshot() os.Error
}

// A checkpoint stored locally in a metadata subdirectory.
type LocalCkpt struct {

	// Representation of the tree and all its contents
	root *fs.Dir

	strong string

	parents []Checkpoint
	
	// Reference to the local checkpoint log which contains this entry
	log *LocalCkptLog

	// Location containing the metadata information 
	// for this checkpoint
	ckptDir string

	// Timestamp when the checkpoint was taken
	tstamp int64
}

// Placeholder Checkpoint internally representing a Null Object.
// A nilCkpt at the head indicates an empty log.
type nilCkpt struct{}

func (nc *nilCkpt) Root() *fs.Dir { return nil }

func (nc *nilCkpt) Parents() []Checkpoint { return []Checkpoint{} }

func (nc *nilCkpt) Strong() string { return "" }

func (nc *nilCkpt) Tstamp() int64 { return int64(0) }

// A checkpoint log that manages and tracks a LocalDirStore over time.
type LocalCkptLog struct {

	// Tracked local directory
	RootPath string
	
	// Strong checksum of the current head
	head string
	
	// Filter used to index the local directory on snapshot.
	Filter fs.IndexFilter
}

// Filter function to exclude the metadata subdirectory from indexing.
func excludeMetadata(path string, f *os.FileInfo) bool {
	_, name := filepath.Split(path)
	return !f.IsDirectory() || name != METADATA_NAME
}

// Same usage as filepath.Join, but path will be relative to the metadata directory.
func (log *LocalCkptLog) metadataPath(parts ...string) string {
	return filepath.Join(append([]string{log.RootPath, METADATA_NAME}, parts...)...)
}

func (log *LocalCkptLog) createFilter() fs.IndexFilter {
	if log.Filter == nil {
		return excludeMetadata
	}
	return func(path string, f *os.FileInfo) bool {
		return log.Filter(path, f) && excludeMetadata(path, f)
	}
}

// Initialize a local checkpoint log and its dir store.
func (log *LocalCkptLog) Init() (err os.Error) {

	refsLocalPath := log.metadataPath("refs", "local")
	err = os.MkdirAll(refsLocalPath, 0755)
	if err != nil {
		debug.PrintStack()
		return err
	}

	headPath := filepath.Join(refsLocalPath, "head")
	headLines, err := readLines(headPath)
	if err == nil && len(headLines) > 0 {
		debug.PrintStack()
		log.head = strings.TrimSpace(headLines[0])
	}

	// Exclude checkpoint log metadata directory from indexing

	return err
}

func (log *LocalCkptLog) Checkpoint(strong string) (Checkpoint, os.Error) {
	ckptDir := log.metadataPath("logs", strong[:2], strong)
	ckpt := &LocalCkpt{log: log, strong: strong, ckptDir: ckptDir}

	_, err := os.Stat(ckptDir)
	if err != nil {
		return &nilCkpt{}, err
	}

	err = ckpt.Init()
	return ckpt, err
}

func (log *LocalCkptLog) Head() (Checkpoint, os.Error) {
	if log.head == "" {
		return new(nilCkpt), nil
	}
	return log.Checkpoint(log.head)
}

func (log *LocalCkptLog) Snapshot() (Checkpoint, os.Error) {
	root := fs.IndexDir(log.RootPath, log.createFilter(), nil)
	ckpt := &LocalCkpt{log: log, root: root}

	sec, nsec, _ := os.Time()
	ckpt.tstamp = sec*1000000000 + nsec

	head, err := log.Head()
	if err == nil && head.Strong() != "" {
		ckpt.parents = append(ckpt.parents, head)
	}

	err = ckpt.create()
	if err != nil {
		return ckpt, err
	}

	err = log.setHead(ckpt.Strong())
	return ckpt, err
}

func (log *LocalCkptLog) setHead(strong string) (err os.Error) {
	err = os.MkdirAll(log.metadataPath("refs", "local"), 0755)
	if err != nil {
		debug.PrintStack()
		return err
	}

	headPath := log.metadataPath("refs", "local", "head")
	err = writeLines(headPath, strong)
	if err != nil {
		debug.PrintStack()
		return err
	}

	log.head = strong
	return err
}

func (ckpt *LocalCkpt) Init() (err os.Error) {
	rootFp, err := os.Open(filepath.Join(ckpt.ckptDir, "root"))
	if err != nil {
		debug.PrintStack()
		return err
	}

	decoder := gob.NewDecoder(rootFp)
	ckpt.root = &fs.Dir{}
	err = decoder.Decode(ckpt.root)
	if err != nil {
		debug.PrintStack()
		return err
	}

	// parse parents, pull from log
	// this will recursively load them
	lines, err := readLines(filepath.Join(ckpt.ckptDir, "checkpoint"))
	if err != nil {
		debug.PrintStack()
		return err
	}

	for linenum, line := range lines {
		fields := strings.Split(line, " ")
		if len(fields) < 2 {
			return os.NewError(fmt.Sprintf(
				"Invalid line %d in checkpoint %s metadata: %s",
				linenum, ckpt.strong, line))
		}

		var parent Checkpoint
		switch fields[0] {
		case "root":
			verifyRoot := fields[1]
			if verifyRoot != ckpt.root.Strong() {
				err = os.NewError(fmt.Sprintf(
					"Inconsistent checkpoint! Expect %s, root was %s",
					verifyRoot, ckpt.root.Strong()))
			}
		case "parent":
			parent, err = ckpt.log.Checkpoint(fields[1])
			if err == nil {
				ckpt.parents = append(ckpt.parents, parent)
			}
		case "tstamp":
			ckpt.tstamp, err = strconv.Atoi64(fields[1])
		default:
			err = os.NewError(fmt.Sprintf(
				"Invalid line %d in checkpoint %s metadata: %s",
				linenum, ckpt.strong, line))
		}

		if err != nil {
			break
		}
	}

	return err
}

// Create this checkpoint in the log that contains it.
func (ckpt *LocalCkpt) create() os.Error {
	strong := ckpt.Strong()
	ckpt.ckptDir = ckpt.log.metadataPath("logs", strong[:2], strong)

	err := os.MkdirAll(ckpt.ckptDir, 0755)
	if err != nil {
		debug.PrintStack()
		return err
	}

	rootFp, err := os.Create(filepath.Join(ckpt.ckptDir, "root"))
	if err != nil {
		debug.PrintStack()
		return err
	}
	defer rootFp.Close()

	encoder := gob.NewEncoder(rootFp)
	err = encoder.Encode(ckpt.root)
	if err != nil {
		debug.PrintStack()
		return err
	}

	ckptFp, err := os.Create(filepath.Join(ckpt.ckptDir, "checkpoint"))
	if err != nil {
		debug.PrintStack()
		return err
	}
	defer ckptFp.Close()
	ckptFp.Write(ckpt.stringBytes())

	return nil
}

// Parent checkpoints from which this checkpoint derives in the log.
func (ckpt *LocalCkpt) Parents() []Checkpoint {
	return ckpt.parents
}

// Time at which the checkpoint was taken.
func (ckpt *LocalCkpt) Tstamp() int64 {
	return ckpt.tstamp
}

// Tree state at the time this checkpoint was taken.
func (ckpt *LocalCkpt) Root() *fs.Dir {
	return ckpt.root
}

// Calculate the strong checksum of a checkpoint.
// This makes the checkpoint content addressable to the state of the tree,
// the time at which the checkpoint was taken, and the lineage of the 
// checkpoint.
func (ckpt *LocalCkpt) Strong() string {
	if ckpt.strong == "" {
		ckpt.strong = ckpt.calcStrong()
	}
	return ckpt.strong
}

// Calculate the strong checksum of a checkpoint.
func (ckpt *LocalCkpt) calcStrong() string {
	var sha1 = sha1.New()
	sha1.Write(ckpt.stringBytes())
	return fs.ToHexString(sha1)
}

// Represent a local checkpoint as a byte string data.
func (ckpt *LocalCkpt) stringBytes() []byte {
	buf := bytes.NewBufferString("")
	fmt.Fprintf(buf, "root %s\n", ckpt.root.Strong())
	fmt.Fprintf(buf, "tstamp %d\n", ckpt.Tstamp())
	for _, parent := range ckpt.parents {
		fmt.Fprintf(buf, "parent %s\n", parent.Strong())
	}
	return buf.Bytes()
}

// Read a file as a slice of strings, one per line.
func readLines(path string) ([]string, os.Error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	result := []string{}
	lineReader, err := bufio.NewReaderSize(f, 80)
	if err != nil {
		return nil, err
	}

	for {
		line, _, err := lineReader.ReadLine()
		if err == os.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		lineStr := strings.TrimSpace(string(line))
		result = append(result, lineStr)
	}

	return result, nil
}

// Write the slice of strings to a file, one per line.
// writeLines... goin thru my mind...
func writeLines(path string, lines ...string) os.Error {
	f, err := os.Create(path)
	if err != nil {
		debug.PrintStack()
		return err
	}
	defer f.Close()

	w, err := bufio.NewWriterSize(f, 80)
	if err != nil {
		debug.PrintStack()
		return err
	}
	defer w.Flush() // defers execute LIFO

	for _, line := range lines {
		_, err = w.WriteString(line)
		if err != nil {
			debug.PrintStack()
			return err
		}
		_, err = w.WriteString("\n")
		if err != nil {
			debug.PrintStack()
			return err
		}
	}

	return nil
}
