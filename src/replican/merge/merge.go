
package merge

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"replican/blocks"
)

type BlockMatch struct {
	SrcBlock *blocks.Block
	DstOffset int64
}

type FileMatch struct {
	SrcSize int64
	DstSize int64
	BlockMatches []*BlockMatch
}

type RangePair struct {
	From int64
	To int64
}

func (r *RangePair) Size() int64 {
	return r.To - r.From
}

func Match(src string, dst string) (match *FileMatch, err os.Error) {
	var srcFile *blocks.File
	srcFile, err = blocks.IndexFile(src)
	if srcFile == nil {
		return nil, err
	}
	
	match, err = MatchFile(srcFile, dst)
	return match, err
}

func MatchFile(srcFile *blocks.File, dst string) (match *FileMatch, err os.Error) {
	srcBlockIndex := blocks.IndexBlocks(srcFile)
	
	match, err = MatchIndex(srcBlockIndex, dst)
	if match != nil {
		match.SrcSize = srcFile.Size()
	}
	
	return match, err
}

func MatchIndex(srcBlockIndex *blocks.BlockIndex, dst string) (match *FileMatch, err os.Error) {
	match = new(FileMatch)
	var dstOffset int64
	
	dstF, err := os.Open(dst)
	if dstF == nil {
		return nil, err
	}
	defer dstF.Close()
	
	if dstInfo, err := dstF.Stat(); dstInfo != nil {
		match.DstSize = dstInfo.Size
	} else {
		return nil, err
	}
	
	dstWeak := new(blocks.WeakChecksum)
	var buf [blocks.BLOCKSIZE]byte
	var sbuf [1]byte
	var window []byte
	
	// Scan a block,
	// then roll checksum a byte at a time until match or eof
	// repeat above until eof
SCAN:
	for {
		switch rd, err := dstF.Read(buf[:]); true {
		case rd < 0:
			return nil, err
			
		case rd == 0:
			break SCAN
		
		case rd > 0:
			blocksize := rd
			dstOffset += int64(rd)
			window = buf[:rd]
			
			dstWeak.Reset()
			dstWeak.Write(window[:])
			
			for {
				// Check for a weak checksum match
				if matchBlock, has := srcBlockIndex.WeakMap[dstWeak.Get()]; has {
					
					// Double-check with the strong checksum
					if blocks.StrongChecksum(window[:blocksize]) == matchBlock.Strong() {
					
						// We've got a block match in dest
						match.BlockMatches = append(match.BlockMatches, &BlockMatch{
							SrcBlock:matchBlock, 
							DstOffset: dstOffset - int64(blocksize) })
						break
					}
				}
				
				// Read the next byte
				switch srd, err := dstF.Read(sbuf[:]); true {
				case srd < 0:
					return nil, err
					
				case srd == 0:
					break SCAN
				
				case srd == 1:
					dstOffset++
					
					// Roll the weak checksum & the buffer
					dstWeak.Roll(window[0], sbuf[0])
					window = append(window[1:], sbuf[0])
					break
				
				case srd > 1:
					return nil, os.NewError("Internal read error trying advance one byte.")
				}
			}
		}
	}
	
	return match, nil
}

func (match *FileMatch) NotMatched() (ranges []*RangePair) {
	start := int64(0)
	
	for _, blockMatch := range match.BlockMatches {
		if start < blockMatch.DstOffset {
			ranges = append(ranges, &RangePair{From:start, To:blockMatch.DstOffset})
		}
		start = blockMatch.DstOffset + int64(blocks.BLOCKSIZE)
	}
	
	if start < match.DstSize {
		ranges = append(ranges, &RangePair{From:start, To:match.DstSize})
	}
	
	return ranges
}

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
	
	Exec(srcStore blocks.BlockStore, dstStore *blocks.LocalStore) os.Error
	
}

// Rename a file. Paths are relative.
type Rename struct {
	From string
	To string
}

func (rename *Rename) Exec(srcStore blocks.BlockStore, dstStore *blocks.LocalStore) os.Error {
	return os.Rename(dstStore.LocalPath(rename.From), dstStore.LocalPath(rename.To))
}

// Keep a file. Yeah, that's right. Just leave it alone.
type Keep struct {
	Path string
}

func (keep *Keep) Exec(srcStore blocks.BlockStore, dstStore *blocks.LocalStore) os.Error { }

// Delete a file. Paths are relative.
type Delete struct {
	Path string
}

func (delete *Delete) Exec(srcStore blocks.BlockStore, dstStore *blocks.LocalStore) os.Error {
	return os.RemoveAll(dstStore.LocalPath(delete.Path))
}

// Set a file to a different size. Paths are relative.
type Resize struct {
	Path string
	Size int64
}

func (resize *Resize) Exec(srcStore blocks.BlockStore, dstStore *blocks.LocalStore) os.Error {
	return os.Truncate(dstStore.LocalPath(resize.Path), resize.Size)
}

// Start a temp file to recieve changes on a local destination file.
// The temporary file is created with specified size and no contents.
type LocalTemp struct {
	LocalPath string
	Size int64
	TmpPath string
}

// Replace the local file with its temporary
type ReplaceWithTemp struct {
	Temp *LocalTemp
}

// Copy a range of data known to already be in the local destination file.
type LocalTmpCopy struct {
	Temp LocalTemp
	LocalFrom int64
	LocalTo int64
	TmpFrom int64
	TmpTo int64
}

//func (drc *DstRangeCopy) Exec(srcStore blocks.BlockStore, dstStore *blocks.LocalStore) os.Error {
//	dst := dstStore.LocalPath(drc.Path)
//	
//}

// Copy a range of data from the source file into a local temp file.
type SrcTmpCopy struct {
	Temp LocalTemp
	SrcPath string
	SrcFrom int64
	SrcTo int64
	TmpFrom int64
	TmpTo int64
}

// Copy a range of data from the source file to the destination file.
type SrcTmpCopy struct {
	Path string
	SrcFrom int64
	SrcTo int64
	DstFrom int64
	DstTo int64
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
		if dstNode, has := dstStore.Index().StrongMap[srcNode.Strong()] {
			dstFsNode, isFsNode := dstNode.(blocks.FsNode)
			dstPath := blocks.RelPath(dstFsNode)
			
			if srcPath != dstPath {
				plan.Cmds = append(plan.Cmds, &Rename{ From:dstPath, To:srcPath })
			} else {
				plan.Cmds = append(plan.Cmds, &Keep{ Path:srcPath })
			}
			
			return false
			
		// If its a file, figure out what to do with it
		} else if (isFile) {
			dstFilePath := dstStore.LocalPath(blocks.RelPath(srcFile))
			
			switch dstFileInfo, err := os.Stat(dstFilePath); true {
			
			// Destination is not a file, so get rid of whatever is there first
			case dstFileInfo != nil && !dstFileInfo.IsRegular():
				plan.Cmds = append(plan.Cmds, &Delete{ Path: srcPath })
				fallthrough
			
			// Destination file does not exist, so full source copy needed
			case dstFileInfo == nil:
				plan.Cmds = append(plan.Cmds, &SrcDstCopy{
					Path: srcPath, 
					SrcFrom: int64(0),
					SrcTo: srcFile.Size(),
					DstFrom: int64(0),
					DstTo: srcFile.Size()})
				break
			
			// Destination file exists, add block-level commands
			default:
				plan.appendFilePlan(srcPath, dstFilePath)
				break
			
			}
		}
		
		return !isFile
	})
	
	return plan 
}

func (plan *PatchPlan) appendFilePlan(srcPath string, dst string) os.Error {
	match, err := MatchIndex(plan.srcStore.Index(), dst)
	if match == nil {
		return err
	}
	
	// Create a local temporary file in which to effect changes
	localTemp := &LocalTemp{ LocalPath: dst, Size: match.SrcSize }
	plan.Cmds = append(plan.Cmds, tmp)
	
	for _, blockMatch := range match.BlockMatches {
		plan.Cmds = append(plan.Cmds, &LocalTmpCopy{
			Temp: localTemp,
			LocalFrom: blockMatch.SrcBlock.Offset(),
			LocalTo: blockMatch.SrcBlock.Offset() + int64(blocks.BLOCKSIZE),
			TmpFrom: blockMatch.DstOffset,
			TmpTo: blockMatch.DstOffset + int64(blocks.BLOCKSIZE)})
	}
	
	for _, srcRange := range match.NotMatched() {
		plan.Cmds = append(plan.Cmds, &SrcTmpCopy{
			Temp: localTemp,
			SrcPath: srcPath,
			SrcFrom: srcRange.From,
			SrcTo: srcRange.To,
			TmpFrom: srcRange.From,
			TmpTo: srcRange.To})
	}
	
	// Replace dst file with temp
	plan.Cmds = append(plan.Cmds, &ReplaceWithTemp{ Temp: localTemp })
	
	return nil
}



