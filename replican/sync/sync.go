package sync

import (
	"os"

	"github.com/cmars/replican-sync/replican/fs"
)

func Sync(src string, dst string) os.Error {
	srcStore, err := fs.NewLocalStore(src)
	if err != nil {
		return err
	}

	dstStore, err := fs.NewLocalStore(dst)
	if err != nil {
		return err
	}

	patchPlan := NewPatchPlan(srcStore, dstStore)
	_, err = patchPlan.Exec()

	return err
}
