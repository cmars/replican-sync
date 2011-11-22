package fs

import (
	"fmt"
	"hash"
	"io"
	"os"
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

// Render a Hash as a hexadecimal string.
func ToHexString(hash hash.Hash) string {
	return fmt.Sprintf("%x", hash.Sum())
}

// Given an absolute path and a common root, return 
// the relative path to the common root.
func MakeRelative(fullpath string, rootpath string) (relpath string) {
	relpath = strings.Replace(fullpath, rootpath, "", 1)
	relpath = strings.TrimLeft(relpath, "/\\")
	return relpath
}
