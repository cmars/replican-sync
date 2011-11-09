package track

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cmars/replican-sync/replican/fs"
)

const METADATA_NAME = ".replican"

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

	// Append a checkpoint of block store
	Append(store fs.BlockStore)
}

type LocalCkpt struct {
	log *LocalCkptLog

	ckptDir string

	strong string
	root   *fs.Dir
	Tstamp int64

	parents []Checkpoint
}

type LocalCkptLog struct {
	RootPath string
}

func (log *LocalCkptLog) Init() os.Error {
	mdPath := filepath.Join(log.RootPath, METADATA_NAME)
	return os.MkdirAll(mdPath, 0755)
}

func (log *LocalCkptLog) Checkpoint(strong string) Checkpoint {
	ckptDir := filepath.Join(log.RootPath, METADATA_NAME, strong)
	ckpt := &LocalCkpt{log: log, strong: strong, ckptDir: ckptDir}
	ckpt.Init()
	return ckpt
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
		if err != nil {
			return nil, err
		}

		lineStr := strings.TrimSpace(string(line))
		result = append(result, lineStr)
	}

	return result, nil
}

func (ckpt *LocalCkpt) Init() (err os.Error) {
	// parse parents, pull from log
	// this will recursively load them
	parentsFile := filepath.Join(ckpt.ckptDir, "parents")
	lines, err := readLines(parentsFile)
	if err != nil {
		return err
	}

READLOOP:
	for linenum, line := range lines {
		fields := strings.Split(line, " ")
		if len(fields) < 2 {
			return os.NewError(fmt.Sprintf(
				"Invalid line %d in checkpoint %s metadata: %s",
				linenum, ckpt.strong, line))
		}

		switch fields[0] {
		case "parent":
			ckpt.parents = append(ckpt.parents, ckpt.log.Checkpoint(fields[1]))
		case "tstamp":
			ckpt.Tstamp, err = strconv.Atoi64(fields[1])
			if err != nil {
				break READLOOP
			}
		default:
			return os.NewError(fmt.Sprintf(
				"Invalid line %d in checkpoint %s metadata: %s",
				linenum, ckpt.strong, line))
		}
	}

	return err
}

func (ckpt *LocalCkpt) Parents() []Checkpoint {
	return ckpt.parents
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
	fmt.Fprintf(buf, "tstamp\t%d\n", ckpt.Tstamp)
	fmt.Fprintf(buf, "root\t%s\n", ckpt.root.Strong())
	for _, parent := range ckpt.parents {
		fmt.Fprintf(buf, "parent\t%s\n", parent.Strong())
	}
	return buf.Bytes()
}

func IndexCheckpoint(path string, f *os.FileInfo) bool {
	_, name := filepath.Split(path)
	return !f.IsDirectory() || name != METADATA_NAME
}
