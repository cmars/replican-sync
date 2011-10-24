
package merge

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"replican/fs"
)

func PatchFile(src string, dst string) os.Error {
	match, err := Match(src, dst)
	if match == nil {
		return err
	}
	
	var buf [fs.BLOCKSIZE]byte
	
	_, dstname := filepath.Split(dst)
	newdstF, err := ioutil.TempFile("", dstname)
	if newdstF == nil { return err }
	defer newdstF.Close()
	
	dstF, err := os.Open(dst)
	if dstF == nil { return err }
	defer dstF.Close()
	
	// Write blocks from dst that we already have
DST_2_NEWDST:
	for _, blockMatch := range match.BlockMatches {
		dstF.Seek(blockMatch.DstOffset, 0)
		newdstF.Seek(blockMatch.DstOffset, 0)
		
		switch rd, err := dstF.Read(buf[:]); true {
		case rd < 0:
			return err
		
		case rd == 0:
			break DST_2_NEWDST
		
		case rd > 0:
			newdstF.Write(buf[:rd])
		}
	}
	
	srcF, err := os.Open(src)
	if srcF == nil { return err }
	defer srcF.Close()
	
	// Fill in the rest from src
SRC_2_NEWDST:
	for _, notMatch := range match.NotMatched() {
		srcF.Seek(notMatch.From, 0)
		newdstF.Seek(notMatch.From, 0)
		
		for toRd := notMatch.Size(); toRd > 0; {
			var rd int
			switch rd, err := srcF.Read(buf[:]); true {
			case rd < 0:
				return err
		
			case rd == 0:
				break SRC_2_NEWDST
		
			case rd > 0:
				newdstF.Write(buf[:rd])
			}
			
			toRd -= int64(rd)
		}
	}
	
	newdst := newdstF.Name()
	
	newdstF.Close()
	dstF.Close()
	
	os.Remove(dst)
	
	err = fs.Move(newdst, dst)
	
	return err
}

type PathRef interface {
	Resolve() string
}

type AbsolutePath string

func (absPath AbsolutePath) Resolve() string {
	return string(absPath)
}

type LocalPath struct {
	LocalStore *fs.LocalStore
	RelPath string
}

func (localPath *LocalPath) String() string {
	return localPath.RelPath
}

func (localPath *LocalPath) Resolve() string {
	return localPath.LocalStore.Resolve(localPath.RelPath)
}

type PatchCmd interface {
	
	String() string
	
	Exec(srcStore fs.BlockStore) os.Error
	
}

func mkParentDirs(path PathRef) (os.Error) {
	dir, _ := filepath.Split(path.Resolve())
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return nil
}

// Copy a local file.
type Transfer struct {
	From *LocalPath
	To *LocalPath
	
	relocRefs map[string]int
}

func (transfer *Transfer) String() string {
	return fmt.Sprintf("Transfer %s to %s", transfer.From, transfer.To)
}

func (transfer *Transfer) Exec(srcStore fs.BlockStore) (err os.Error) {
	transfer.relocRefs[transfer.From.RelPath]--
	refCount := transfer.relocRefs[transfer.From.RelPath]
	
	switch {
	case refCount == 0:
		return transfer.move(srcStore)
	case refCount > 0:
		return transfer.copy(srcStore)
	}
	
	return os.NewError(fmt.Sprintf(
		"Cannot transfer %s: reference count underflow", transfer.From.RelPath))
}

func (transfer *Transfer) copy(srcStore fs.BlockStore) os.Error {
	if err := mkParentDirs(transfer.To); err != nil { return err }
	
	srcF, err := os.Open(transfer.From.Resolve())
	if err != nil { return err }
	defer srcF.Close()
	
	dstF, err := os.Create(transfer.To.Resolve())
	if err != nil { return err }
	defer dstF.Close()
	
	_, err = io.Copy(dstF, srcF)
	return err
}

func (transfer *Transfer) move(srcStore fs.BlockStore) os.Error {
	if err := mkParentDirs(transfer.To); err != nil { return err }
	
	return fs.Move(transfer.From.Resolve(), transfer.To.Resolve())
}

// Keep a file. Yeah, that's right. Just leave it alone.
type Keep struct {
	Path PathRef
}

func (keep *Keep) String() string {
	return fmt.Sprintf("Keep %s", keep.Path.Resolve())
}

func (keep *Keep) Exec(srcStore fs.BlockStore) os.Error { 
	return nil
}

// Register a conflict
type Conflict struct {
	Path *LocalPath
	FileInfo *os.FileInfo
	
	relocPath string
}

func (conflict *Conflict) String() string {
	return fmt.Sprintf("Conflict found at %s, redirecting...", conflict.Path)
}

func (conflict *Conflict) Exec(srcStore fs.BlockStore) (err os.Error) {
	conflict.relocPath, err = conflict.Path.LocalStore.Relocate(conflict.Path.RelPath)
	return err
}

func (conflict *Conflict) Cleanup() os.Error {
	return os.RemoveAll(conflict.relocPath)
}

// Set a file to a different size. Paths are relative.
type Resize struct {
	Path PathRef
	Size int64
}

func (resize *Resize) String() string {
	return fmt.Sprintf("Resize %s to %d bytes", resize.Path, resize.Size)
}

func (resize *Resize) Exec(srcStore fs.BlockStore) os.Error {
	return os.Truncate(resize.Path.Resolve(), resize.Size)
}

// Start a temp file to recieve changes on a local destination file.
// The temporary file is created with specified size and no contents.
type LocalTemp struct {
	Path PathRef
	Size int64
	
	localFh *os.File
	tempFh *os.File
}

func (localTemp *LocalTemp) String() string {
	return fmt.Sprintf("Create a temporary file for %s, size=%d bytes", localTemp.Path, localTemp.Size)
}

func (localTemp *LocalTemp) Exec(srcStore fs.BlockStore) (err os.Error) {
	localTemp.localFh, err = os.Open(localTemp.Path.Resolve())
	if localTemp.localFh == nil { return err }
	
	localDir, localName := filepath.Split(localTemp.Path.Resolve())
	
	localTemp.tempFh, err = ioutil.TempFile(localDir, localName)
	if localTemp.tempFh == nil { return err }
	
	err = localTemp.tempFh.Truncate(localTemp.Size)
	if (err != nil) { return err }
	
	return nil
}

// Replace the local file with its temporary
type ReplaceWithTemp struct {
	Temp *LocalTemp
}

func (rwt *ReplaceWithTemp) String() string {
	return fmt.Sprintf("Replace %s with the temporary backup", rwt.Temp.Path)
}

func (rwt *ReplaceWithTemp) Exec(srcStore fs.BlockStore) (err os.Error) {
	tempName := rwt.Temp.tempFh.Name()
	rwt.Temp.localFh.Close()
	rwt.Temp.localFh = nil
	
	rwt.Temp.tempFh.Close()
	rwt.Temp.tempFh = nil
	
	err = os.Remove(rwt.Temp.Path.Resolve())
	if err != nil { return err }
	
	err = fs.Move(tempName, rwt.Temp.Path.Resolve())
	if err != nil { return err }
	
	return nil
}

// Copy a range of data known to already be in the local destination file.
type LocalTempCopy struct {
	Temp *LocalTemp
	LocalOffset int64
	TempOffset int64
	Length int64
}

func (ltc *LocalTempCopy) String() string {
	return fmt.Sprintf("Copy %d bytes from offset %d in target file %s to offset %d in temporary file",
		ltc.Length, ltc.LocalOffset, ltc.Temp.Path, ltc.TempOffset)
}

func (ltc *LocalTempCopy) Exec(srcStore fs.BlockStore) (err os.Error) {
	_, err = ltc.Temp.localFh.Seek(ltc.LocalOffset, 0)
	if err != nil { return err }
	
	_, err = ltc.Temp.tempFh.Seek(ltc.TempOffset, 0)
	if err != nil { return err }
	
	_, err = io.Copyn(ltc.Temp.tempFh, ltc.Temp.localFh, ltc.Length)
	return err
}

// Copy a range of data from the source file into a local temp file.
type SrcTempCopy struct {
	Temp *LocalTemp
	SrcStrong string
	SrcOffset int64
	TempOffset int64
	Length int64
}

func (stc *SrcTempCopy) String() string {
	return fmt.Sprintf("Copy %d bytes from offset %d from source %s to offset %d in temporary file",
		stc.Length, stc.SrcOffset, stc.SrcStrong, stc.TempOffset)
}

func (stc *SrcTempCopy) Exec(srcStore fs.BlockStore) os.Error {
	stc.Temp.tempFh.Seek(stc.TempOffset, 0)
	return srcStore.ReadInto(stc.SrcStrong, stc.SrcOffset, stc.Length, stc.Temp.tempFh)
}

// Copy a range of data from the source file to the destination file.
type SrcFileDownload struct {
	SrcFile *fs.File
	Path PathRef
	Length int64
}

func (sfd *SrcFileDownload) String() string {
	return fmt.Sprintf("Copy entire source %s to %s", sfd.SrcFile.Strong(), sfd.Path)
}

func (sfd *SrcFileDownload) Exec(srcStore fs.BlockStore) os.Error {
	if err := mkParentDirs(sfd.Path); err != nil { return err }
	
	dstFh, err := os.Create(sfd.Path.Resolve())
	if dstFh == nil { return err }
	
	return srcStore.ReadInto(sfd.SrcFile.Strong(), 0, sfd.SrcFile.Size, dstFh)
}

type PatchPlan struct {
	Cmds []PatchCmd
	
	srcStore fs.BlockStore
	dstStore *fs.LocalStore
}

func NewPatchPlan(srcStore fs.BlockStore, dstStore *fs.LocalStore) *PatchPlan {
	plan := &PatchPlan{srcStore: srcStore, dstStore: dstStore}
	
	relocRefs := make(map[string]int)
	
	// Find all the FsNode matches
	fs.Walk(srcStore.Root(), func(srcNode fs.Node) bool {
		
		// Ignore non-FsNodes
		srcFsNode, isSrcFsNode := srcNode.(fs.FsNode)
		if !isSrcFsNode {
			return false
		}
		
		srcFile, isSrcFile := srcNode.(*fs.File)
		srcPath := fs.RelPath(srcFsNode)
		
		dstNode, hasDstNode := dstStore.Index().StrongFsNode(srcNode.Strong())
		
		isDstFile := false
		if hasDstNode {
			_, isDstFile = dstNode.(*fs.File)
		}
		
		dstFilePath := dstStore.Resolve(srcPath)
		dstFileInfo, _ := os.Stat(dstFilePath)
		
//		fmt.Printf("srcPath=%s hasDstNode=%v isDstFsNode=%v isSrcFile=%v, isDstFile=%v\n%v\n\n",
//			srcPath, hasDstNode, isDstFsNode, isSrcFile, isDstFile, dstNode)
		
		// Resolve dst node that matches strong checksum with source
		if hasDstNode && isSrcFile == isDstFile {
			dstPath := fs.RelPath(dstNode)
			relocRefs[dstPath]++ // dstPath will be used in this cmd, inc ref count
			
			if srcPath != dstPath {
				// Local dst file needs to be renamed or copied to src path
				from := &LocalPath{ LocalStore: dstStore, RelPath: dstPath }
				to := &LocalPath{ LocalStore: dstStore, RelPath: srcPath }
				plan.Cmds = append(plan.Cmds, 
					&Transfer{ From: from, To: to, relocRefs: relocRefs })
			} else {
				// Same path, keep it where it is
				plan.Cmds = append(plan.Cmds, &Keep{
					Path: &LocalPath{ LocalStore: dstStore, RelPath: srcPath }})
			}
			
		// If its a file, figure out what to do with it
		} else if (isSrcFile) {
			
			switch {
			
			// Destination is not a file, so get rid of whatever is there first
			case dstFileInfo != nil && !dstFileInfo.IsRegular():
				plan.Cmds = append(plan.Cmds, &Conflict{ 
					Path: &LocalPath{ LocalStore: dstStore, RelPath: srcPath }, 
					FileInfo: dstFileInfo })
				fallthrough
			
			// Destination file does not exist, so full source copy needed
			case dstFileInfo == nil:
				plan.Cmds = append(plan.Cmds, &SrcFileDownload{
					SrcFile: srcFile,
					Path: &LocalPath{ LocalStore: dstStore, RelPath: srcPath }})
				break
			
			// Destination file exists, add block-level commands
			default:
				plan.appendFilePlan(srcFile, srcPath)
				break
			}
			
		// If its a directory, check for conflicting files of same name
		} else {
			
			if dstFileInfo != nil && !dstFileInfo.IsDirectory() {
				plan.Cmds = append(plan.Cmds, &Conflict{ 
					Path: &LocalPath{ LocalStore: dstStore, RelPath: dstFilePath },
					FileInfo: dstFileInfo })
			}
		}
		
		return !isSrcFile
	})
	
	return plan 
}

func (plan *PatchPlan) Exec() (failedCmd PatchCmd, err os.Error) {
	conflicts := []*Conflict{}
	for _, cmd := range plan.Cmds {
		err = cmd.Exec(plan.srcStore)
		if err != nil {
			return cmd, err
		}
		
		if conflict, is := cmd.(*Conflict); is {
			conflicts = append(conflicts, conflict)
		}
	}
	
	for _, conflict := range conflicts {
		conflict.Cleanup()
	}
	
	return nil, nil
}

func (plan *PatchPlan) String() string {
	buf := &bytes.Buffer{}
	for _, cmd := range plan.Cmds {
		fmt.Fprintf(buf, "%v\n", cmd)
	}
	return string(buf.Bytes())
}

func (plan *PatchPlan) appendFilePlan(srcFile *fs.File, dstPath string) os.Error {
	match, err := MatchIndex(plan.srcStore.Index(), plan.dstStore.Resolve(dstPath))
	if match == nil {
		return err
	}
	match.SrcSize = srcFile.Size
	
	// Create a local temporary file in which to effect changes
	localTemp := &LocalTemp{ 
		Path: &LocalPath{ 
			LocalStore: plan.dstStore,
			RelPath: dstPath },
		Size: match.SrcSize }
	plan.Cmds = append(plan.Cmds, localTemp)
	
	for _, blockMatch := range match.BlockMatches {
		plan.Cmds = append(plan.Cmds, &LocalTempCopy{
			Temp: localTemp,
			LocalOffset: blockMatch.SrcBlock.Offset(),
			TempOffset: blockMatch.DstOffset,
			Length: int64(fs.BLOCKSIZE)})
	}
	
	for _, srcRange := range match.NotMatched() {
		plan.Cmds = append(plan.Cmds, &SrcTempCopy{
			Temp: localTemp,
			SrcStrong: srcFile.Strong(),
			SrcOffset: srcRange.From,
			TempOffset: srcRange.From,
			Length: srcRange.To - srcRange.From})
	}
	
	// Replace dst file with temp
	plan.Cmds = append(plan.Cmds, &ReplaceWithTemp{ Temp: localTemp })
	
	return nil
}



