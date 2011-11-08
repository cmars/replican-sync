package track

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cmars/replican-sync/replican/fs"
)

const METADATA_DIR_NAME = ".replican"

type Checkpoint interface {
	Root() *fs.Dir

	Parents() []Checkpoint

	Strong() string
}

type CheckpointLog interface {

	// Fetch the checkpoint with given strong checksum
	Checkpoint(strong string) Checkpoint

	// Fetch the current head checkpoint
	Head() Checkpoint

	// Append a new head checkpoint
	AppendHead(checkpoint Checkpoint)
}

type LocalCkpt struct {
	log *LocalCkptLog

	strong string
	root   *fs.Dir
	Tstamp int64

	Parents []Checkpoint
}

type LocalCkptLog struct {
	RootPath string
}

func (log *LocalCkptLog) Checkpoint(strong string) Checkpoint {
	panic("not impl yet")
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
	fmt.Fprintf(buf, "tstamp\t%d\n", ckpt.Tstamp)
	fmt.Fprintf(buf, "root\t%s\n", ckpt.root.Strong())
	for _, parent := range ckpt.Parents {
		fmt.Fprintf(buf, "parent\t%s\n", parent.Strong())
	}
	return buf.Bytes()
}

func IndexCheckpoint(path string, f *os.FileInfo) bool {
	_, name := filepath.Split(path)
	return !f.IsDirectory() || name != METADATA_DIR_NAME
}
