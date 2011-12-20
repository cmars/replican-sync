package fs

import (
	"io"
	//	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
)

func SplitNames(path string) []string {
	if path == "" {
		return []string{}
	}
	return strings.Split(path, string(os.PathSeparator))
}

// Move src to dst.
// Try a rename. If that fails due to different filesystems,
// try a copy/delete instead.
func Move(src string, dst string) (err os.Error) {
	if _, err = os.Stat(dst); err == nil {
		os.Remove(dst)
	}

	if err = os.Rename(src, dst); err != nil {
		linkErr, isLinkErr := err.(*os.LinkError)
		if !isLinkErr {
			return err
		}

		if causeErr, isErrno := linkErr.Error.(os.Errno); isErrno && causeErr == syscall.EXDEV {
			srcF, err := os.Open(src)
			if err != nil {
				return err
			}
			defer srcF.Close()

			dstF, err := os.Create(dst)
			if err != nil {
				return err
			}
			defer dstF.Close()

			_, err = io.Copy(dstF, srcF)
			if err != nil {
				return err
			}

			srcF.Close()
			err = os.Remove(src)

			return err
		}

		return err
	}

	return nil
}

type postNode struct {
	path string
	info *os.FileInfo
	seen bool
}

type sortFileInfo struct {
	infos *[]os.FileInfo
}

func (sfi *sortFileInfo) Len() int {
	return len(*sfi.infos)
}

func (sfi *sortFileInfo) Less(i, j int) bool {
	l := (*sfi.infos)[i]
	r := (*sfi.infos)[j]
	if l.IsDirectory() == r.IsDirectory() {
		return (*sfi.infos)[i].Name > (*sfi.infos)[j].Name
	}
	return r.IsDirectory()
}

func (sfi *sortFileInfo) Swap(i, j int) {
	(*sfi.infos)[i], (*sfi.infos)[j] = (*sfi.infos)[j], (*sfi.infos)[i]
}

func PostOrderWalk(path string, visitor filepath.Visitor, errors chan<- os.Error) {
	stack := []*postNode{}
	var cur *postNode

	if info, err := os.Stat(path); err == nil {
		cur = &postNode{path: path, info: info, seen: false}
		stack = append(stack, cur)
	} else if errors != nil {
		errors <- err
		return
	}

	for l := len(stack); l > 0; l = len(stack) {
		cur := stack[l-1]

		if cur.seen {
			stack = stack[0 : l-1]
			if cur.info.IsDirectory() {
				visitor.VisitDir(cur.path, cur.info)
			} else {
				visitor.VisitFile(cur.path, cur.info)
			}
			continue
		}

		if cur.info.IsDirectory() {
			if f, err := os.Open(cur.path); err == nil {
				if infos, err := f.Readdir(0); err == nil {
					sfi := &sortFileInfo{infos: &infos}
					sort.Sort(sfi)
					for _, info := range *sfi.infos {
						//						log.Printf("push %s", info.Name)
						stack = append(stack, &postNode{
							path: filepath.Join(cur.path, info.Name),
							info: &info,
							seen: false})
					}
				} else if errors != nil {
					errors <- err
				}
			}
			cur.seen = true
		} else {
			stack = stack[0 : l-1]
			visitor.VisitFile(cur.path, cur.info)
		}
	}
}
