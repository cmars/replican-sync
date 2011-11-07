package track

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"github.com/cmars/replican-sync/replican/fs"
)

type Checkpoint struct {
	strong string
	root   *fs.Dir
	Tstamp int64

	Parents []*Checkpoint
}

func (ckpt *Checkpoint) Strong() string {
	if ckpt.strong == "" {
		ckpt.strong = ckpt.calcStrong()
	}
	return ckpt.strong
}

// Calculate the strong checksum of a checkpoint.
func (ckpt *Checkpoint) calcStrong() string {
	var sha1 = sha1.New()
	sha1.Write(ckpt.stringBytes())
	return fs.ToHexString(sha1)
}

func (ckpt *Checkpoint) stringBytes() []byte {
	buf := bytes.NewBufferString("")
	fmt.Fprintf(buf, "root\t%s\n", ckpt.root.Strong())
	fmt.Fprintf(buf, "tstamp\t%d\n", ckpt.Tstamp)
	return buf.Bytes()
}

func (ckpt *Checkpoint) Parent() fs.Node {
	if ckpt.Parents != nil && len(ckpt.Parents) > 0 {
		return ckpt.Parents[0]
	}
	return nil
}

func (ckpt *Checkpoint) Root() *fs.Dir {
	return ckpt.root
}
