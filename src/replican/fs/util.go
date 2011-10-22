
package fs

import (
	"io"
	"os"
	"syscall"
)

// Move src to dst.
// Try a rename. If that fails due to different filesystems,
// try a copy/delete instead.
func Move(src string, dst string) os.Error {
	if err := os.Rename(src, dst); err != nil {
		linkErr, isLinkErr := err.(*os.LinkError)
		if !isLinkErr { return err }
		
		if causeErr, isErrno := linkErr.Error.(os.Errno); 
				isErrno && causeErr == syscall.EXDEV {
			srcF, err := os.Open(src)
			if err != nil { return err }
			defer srcF.Close()
			
			dstF, err := os.Create(dst)
			if err != nil { return err }
			defer dstF.Close()
			
			_, err = io.Copy(dstF, srcF)
			if err != nil { return err }
			
			srcF.Close()
			err = os.Remove(src)
			
			return err
		}
		
		return err
	}
	
	return nil
}

