package replican

import (
	"os"
	"path/filepath"
	"sort"
)

type postNode struct {
	path string
	info os.FileInfo
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
	if l.IsDir() == r.IsDir() {
		return (*sfi.infos)[i].Name() > (*sfi.infos)[j].Name()
	}
	return r.IsDir()
}

func (sfi *sortFileInfo) Swap(i, j int) {
	(*sfi.infos)[i], (*sfi.infos)[j] = (*sfi.infos)[j], (*sfi.infos)[i]
}

func PostOrderWalk(path string, visitor filepath.WalkFunc, errors chan<- error) {
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
			visitor(cur.path, cur.info, nil)
			continue
		}

		if cur.info.IsDir() {
			if f, err := os.Open(cur.path); err == nil {
				if infos, err := f.Readdir(0); err == nil {
					sfi := &sortFileInfo{infos: &infos}
					sort.Sort(sfi)
					for _, info := range *sfi.infos {
						//						log.Printf("push %s", info.Name)
						stack = append(stack, &postNode{
							path: filepath.Join(cur.path, info.Name()),
							info: info,
							seen: false})
					}
				} else if errors != nil {
					errors <- err
				}
			}
			cur.seen = true
		} else {
			stack = stack[0 : l-1]
			visitor(cur.path, cur.info, nil)
		}
	}
}
