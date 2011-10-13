
package merge

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"replican/blocks"
)

func PatchFile(src string, dst string) os.Error {
	match, err := Match(src, dst)
	if match == nil {
		return err
	}
	
	var buf [blocks.BLOCKSIZE]byte
	
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
	os.Rename(newdst, dst)
	
	return nil
}

type PatchCmd interface {
	
	String() string
	
	Exec(srcStore blocks.BlockStore) os.Error
	
}

// Rename a file.
type Rename struct {
	From string
	To string
}

func (rename *Rename) String() string {
	return fmt.Sprintf("Rename %s to %s", rename.From, rename.To)
}

func (rename *Rename) Exec(srcStore blocks.BlockStore) os.Error {
	return os.Rename(rename.From, rename.To)
}

// Keep a file. Yeah, that's right. Just leave it alone.
type Keep struct {
	Path string
}

func (keep *Keep) String() string {
	return fmt.Sprintf("Keep %s", keep.Path)
}

func (keep *Keep) Exec(srcStore blocks.BlockStore) os.Error { return nil }

// Delete a file.
type Delete struct {
	Path string
}

func (delete *Delete) String() string {
	return fmt.Sprintf("Delete %s", delete.Path)
}

func (delete *Delete) Exec(srcStore blocks.BlockStore) os.Error {
	return os.RemoveAll(delete.Path)
}

// Set a file to a different size. Paths are relative.
type Resize struct {
	Path string
	Size int64
}

func (resize *Resize) String() string {
	return fmt.Sprintf("Resize %s to %d bytes", resize.Path, resize.Size)
}

func (resize *Resize) Exec(srcStore blocks.BlockStore) os.Error {
	return os.Truncate(resize.Path, resize.Size)
}

// Start a temp file to recieve changes on a local destination file.
// The temporary file is created with specified size and no contents.
type LocalTemp struct {
	Path string
	Size int64
	
	localFh *os.File
	tempFh *os.File
}

func (localTemp *LocalTemp) String() string {
	return fmt.Sprintf("Create a temporary file for %s, size=%d bytes", localTemp.Path, localTemp.Size)
}

func (localTemp *LocalTemp) Exec(srcStore blocks.BlockStore) (err os.Error) {
	localTemp.localFh, err = os.Open(localTemp.Path)
	if localTemp.localFh == nil { return err }
	
	localDir, localName := filepath.Split(localTemp.Path)
	
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

func (rwt *ReplaceWithTemp) Exec(srcStore blocks.BlockStore) (err os.Error) {
	tempName := rwt.Temp.tempFh.Name()
	rwt.Temp.localFh.Close()
	rwt.Temp.localFh = nil
	
	rwt.Temp.tempFh.Close()
	rwt.Temp.tempFh = nil
	
	err = os.Remove(rwt.Temp.Path)
	if err != nil { return err }
	
	err = os.Rename(tempName, rwt.Temp.Path)
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

func (ltc *LocalTempCopy) Exec(srcStore blocks.BlockStore) (err os.Error) {
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

func (stc *SrcTempCopy) Exec(srcStore blocks.BlockStore) os.Error {
	stc.Temp.tempFh.Seek(stc.TempOffset, 0)
	return srcStore.ReadInto(stc.SrcStrong, stc.SrcOffset, stc.Length, stc.Temp.tempFh)
}

// Copy a range of data from the source file to the destination file.
type SrcFileDownload struct {
	SrcFile *blocks.File
	Path string
	Length int64
}

func (sfd *SrcFileDownload) String() string {
	return fmt.Sprintf("Copy entire source %s to %s", sfd.SrcFile.Strong(), sfd.Path)
}

func (sfd *SrcFileDownload) Exec(srcStore blocks.BlockStore) os.Error {
	dstFh, err := os.Create(sfd.Path)
	if dstFh == nil { return err }
	
	return srcStore.ReadInto(sfd.SrcFile.Strong(), 0, sfd.SrcFile.Size, dstFh)
}

type PatchPlan struct {
	SrcRoot *blocks.Dir
	DstRoot *blocks.Dir
	Cmds []PatchCmd
	
	srcStore blocks.BlockStore
	dstStore blocks.BlockStore
}

func NewPatchPlan(srcStore blocks.BlockStore, dstStore *blocks.LocalStore) *PatchPlan {
	plan := &PatchPlan{SrcRoot: srcStore.Root(), DstRoot: dstStore.Root()}
	plan.srcStore = srcStore
	plan.dstStore = dstStore
	
	// Find all the FsNode matches
	blocks.Walk(srcStore.Root(), func(srcNode blocks.Node) bool {
		
		// Ignore non-FsNodes
		srcFsNode, isFsNode := srcNode.(blocks.FsNode)
		if !isFsNode {
			return false
		}
		
		srcFile, isFile := srcNode.(*blocks.File)
		srcPath := blocks.RelPath(srcFsNode)
		
		// Try to match at the file level. Might be a Rename or leave in place
		if dstNode, has := dstStore.Index().StrongMap[srcNode.Strong()]; has {
			
			if dstFsNode, isFsNode := dstNode.(blocks.FsNode); isFsNode {
				dstPath := blocks.RelPath(dstFsNode)
			
				if srcPath != dstPath {
					plan.Cmds = append(plan.Cmds, &Rename{ 
						From: dstStore.LocalPath(dstPath),
						To: dstStore.LocalPath(srcPath) })
				} else {
					plan.Cmds = append(plan.Cmds, &Keep{
						Path: dstStore.LocalPath(srcPath) })
				}
			}
						
			return false
			
		// If its a file, figure out what to do with it
		} else if (isFile) {
			dstFilePath := dstStore.LocalPath(blocks.RelPath(srcFile))
			
			switch dstFileInfo, _ := os.Stat(dstFilePath); true {
			
			// Destination is not a file, so get rid of whatever is there first
			case dstFileInfo != nil && !dstFileInfo.IsRegular():
				plan.Cmds = append(plan.Cmds, &Delete{ Path: srcPath })
				fallthrough
			
			// Destination file does not exist, so full source copy needed
			case dstFileInfo == nil:
				plan.Cmds = append(plan.Cmds, &SrcFileDownload{
					SrcFile: srcFile,
					Path: dstFilePath})
				break
			
			// Destination file exists, add block-level commands
			default:
				plan.appendFilePlan(srcFile, dstFilePath)
				break
			
			}
		}
		
		return !isFile
	})
	
	return plan 
}

func (plan *PatchPlan) Exec() (failedCmd PatchCmd, err os.Error) {
	for _, cmd := range plan.Cmds {
		err = cmd.Exec(plan.srcStore)
		if err != nil {
			return cmd, err
		}
	}
	return nil, nil
}

func (plan *PatchPlan) appendFilePlan(srcFile *blocks.File, dst string) os.Error {
	match, err := MatchIndex(plan.srcStore.Index(), dst)
	if match == nil {
		return err
	}
	match.SrcSize = srcFile.Size
	
	// Create a local temporary file in which to effect changes
	localTemp := &LocalTemp{ Path: dst, Size: match.SrcSize }
	plan.Cmds = append(plan.Cmds, localTemp)
	
	for _, blockMatch := range match.BlockMatches {
		plan.Cmds = append(plan.Cmds, &LocalTempCopy{
			Temp: localTemp,
			LocalOffset: blockMatch.SrcBlock.Offset(),
			TempOffset: blockMatch.DstOffset,
			Length: int64(blocks.BLOCKSIZE)})
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



