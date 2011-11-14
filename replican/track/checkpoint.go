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

// A Checkpoint represents the state of a BlockStore's tree
// at a single point in time.
type Checkpoint interface {

	// Representation of the tree and all its contents.
	Root() *fs.Dir

	// Parent checkpoints which preceded this state of the tree.
	// Every checkpoint, except the first, will have at least one parent.
	// A checkpoint that results from a patch operation will have multiple parents.
	Parents() []Checkpoint

	// Strong checksum of the checkpoint.
	// This is a function of the tree checksum, timestamp, and lineage.
	Strong() string

	// When the checkpoint was taken.
	Tstamp() int64
}

// A Checkpoint log tracks all the checkpoints taken of a BlockStore
// over time.
type CheckpointLog interface {

	// Fetch the checkpoint with given strong checksum
	Checkpoint(strong string) (Checkpoint, os.Error)

	// Fetch the current head checkpoint.
	Head() (Checkpoint, os.Error)

	// Create a checkpoint of the current block store state 
	// and append to the head of the log. 
	Snapshot() os.Error

	// The store tracked by this log
	Store() fs.BlockStore
}

// A checkpoint stored locally in a tracked BlockStore.
type LocalCkpt struct {

	// Reference to the local checkpoint log which contains this entry
	log *LocalCkptLog

	// Location containing the metadata information 
	// for this checkpoint
	ckptDir string

	strong string

	// Representation of the tree and all its contents
	root *fs.Dir

	// Timestamp when the checkpoint was taken
	tstamp int64

	parents []Checkpoint
}

// A Null Object instance of a Checkpoint.
type nilCkpt struct{}

func (nc *nilCkpt) Root() *fs.Dir { return nil }

func (nc *nilCkpt) Parents() []Checkpoint { return []Checkpoint{} }

func (nc *nilCkpt) Strong() string { return "" }

func (nc *nilCkpt) Tstamp() int64 { return int64(0) }

// A checkpoint log that manages and tracks a LocalDirStore over time.
type LocalCkptLog struct {

	// Local store directory
	RootPath string

	head string

	Filter fs.IndexFilter

	// Local directory store
	store *fs.LocalDirStore
}

// Filter function to exclude the metadata subdirectory from indexing.
func excludeMetadata(path string, f *os.FileInfo) bool {
	_, name := filepath.Split(path)
	return !f.IsDirectory() || name != METADATA_NAME
}

func (log *LocalCkptLog) metadataPath(parts ...string) string {
	return filepath.Join(append([]string{log.RootPath, METADATA_NAME}, parts...)...)
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

	log.store = &fs.LocalDirStore{
		LocalInfo: &fs.LocalInfo{RootPath: log.RootPath}}

	// Exclude checkpoint log metadata directory from indexing
	if log.Filter == nil {
		log.store.Filter = excludeMetadata
	} else {
		log.store.Filter = func(path string, f *os.FileInfo) bool {
			return log.Filter(path, f) && excludeMetadata(path, f)
		}
	}

	err = log.store.Init()
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
	root := fs.IndexDir(log.RootPath, excludeMetadata, nil)
	ckpt := &LocalCkpt{log: log, root: root}

	sec, nsec, _ := os.Time()
	ckpt.tstamp = sec*1000000000 + nsec

	head, err := log.Head()
	if err == nil && head.Strong() != "" {
		ckpt.parents = append(ckpt.parents, head)
	}

	err = ckpt.Create()
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

func (log *LocalCkptLog) Store() fs.BlockStore {
	return log.store
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

func (ckpt *LocalCkpt) Create() os.Error {
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

func (ckpt *LocalCkpt) Parents() []Checkpoint {
	return ckpt.parents
}

func (ckpt *LocalCkpt) Tstamp() int64 {
	return ckpt.tstamp
}

func (ckpt *LocalCkpt) Root() *fs.Dir {
	return ckpt.root
}

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

func (ckpt *LocalCkpt) stringBytes() []byte {
	buf := bytes.NewBufferString("")
	fmt.Fprintf(buf, "root %s\n", ckpt.root.Strong())
	fmt.Fprintf(buf, "tstamp %d\n", ckpt.Tstamp())
	for _, parent := range ckpt.parents {
		fmt.Fprintf(buf, "parent %s\n", parent.Strong())
	}
	return buf.Bytes()
}

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
