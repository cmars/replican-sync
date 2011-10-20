
package merge

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"replican/blocks"
	"replican/treegen"
	"strings"
	"testing"
	
	"github.com/bmizerany/assert"
)

// Print a description of the steps that the patch plan will follow.
func printPlan(plan *PatchPlan) {
	for i := 0; i < len(plan.Cmds); i++ {
		fmt.Printf("%s\n", plan.Cmds[i].String())
	}
}

// Test that the matcher matches all blocks in two identical files.
func TestMatchIdentity(t *testing.T) {
	srcPath := "./testroot/My Music/0 10k 30.mp4"
	dstPath := srcPath
	
	match, err := Match(srcPath, dstPath)
	
	assert.T(t, err == nil)
	
	nMatches := 0
	for i, match := range match.BlockMatches {
		assert.Equalf(t, int64(0), match.DstOffset % int64(blocks.BLOCKSIZE), 
				"Destination match block# %d not aligned with blocksize! (offset=%d)",
				i, match.DstOffset)
		nMatches++
	}
	
	fileInfo, err := os.Stat(srcPath)
	if fileInfo == nil {
		t.Fatalf("Cannot stat file %s: ", err.String())
	} else {
		nExpectMatches := fileInfo.Size / int64(blocks.BLOCKSIZE)
		if fileInfo.Size % int64(blocks.BLOCKSIZE) > 0 {
			nExpectMatches++
		}
		
		assert.Equal(t, nExpectMatches, int64(nMatches))
	}
	
	lastBlockSize := fileInfo.Size - int64(match.BlockMatches[14].DstOffset)
	assert.Equalf(t, int64(5419), lastBlockSize,
			"Unxpected last block size: %d", lastBlockSize)
}

// Test that the matcher matches blocks properly between two different files.
// The munged file has a few bytes changed at known offsets which we check for.
func TestMatchMunge(t *testing.T) {
	srcPath := "./testroot/My Music/0 10k 30.mp4"
	dstPath := "./testroot/My Music/0 10k 30 munged.mp4"
	
	match, err := Match(srcPath, dstPath)
	
	assert.T(t, err == nil)
	
	nMatches := 0
	for i, match := range match.BlockMatches {
		assert.Equalf(t, int64(0), match.DstOffset % int64(blocks.BLOCKSIZE),
				"Destination match block# %d not aligned with blocksize! (offset=%d)",
				i, match.DstOffset)
		nMatches++
	}
	
	assert.Equal(t, 13, nMatches)
	
	notMatches := match.NotMatched()
	assert.Equal(t, 2, len(notMatches))
}

// Test some corner cases of FileMatch.NotMatched to correctly 
// identify unmatched ranges between two files. No files were harmed 
// in the creation of this test, we're fabricating a fake FileMatch.
func TestHoles(t *testing.T) {
	testMatch := &FileMatch{ 
		SrcSize:99099, DstSize:99099, 
		BlockMatches: []*BlockMatch{
			&BlockMatch{DstOffset:123},
			&BlockMatch{DstOffset:9991},
			&BlockMatch{DstOffset:23023},
		}}
	
	notMatched := testMatch.NotMatched()
	
	assert.Tf(t, len(notMatched) > 0, "Failed to detect obvious holes in match")
	
	for i, unmatch := range(notMatched) {
		switch i {
		case 0:
			assert.Equal(t, int64(0), unmatch.From)
			assert.Equal(t, int64(123), unmatch.To)
			break
			
		case 1:
			assert.Equal(t, int64(8315), unmatch.From)
			assert.Equal(t, int64(9991), unmatch.To)
			break
			
		case 2:
			assert.Equal(t, int64(18183), unmatch.From)
			assert.Equal(t, int64(23023), unmatch.To)
			break
			
		case 3:
			assert.Equal(t, int64(31215), unmatch.From)
			assert.Equal(t, int64(99099), unmatch.To)
			break
		
		default:
			t.Fatalf("Unexpected not-match %v", unmatch)
		}
	}
}

// Test an actual file patch on the munged file scenario from TestMatchMunge.
// Resulting patched file should be identical to the source file.
func TestPatch(t *testing.T) {
	srcPath := "testroot/My Music/0 10k 30.mp4"
	dstPath := "/var/tmp/foo.mp4"
	
	os.Remove(dstPath)
	
	func(){
		origDstF, err := os.Open("testroot/My Music/0 10k 30 munged.mp4")
		assert.T(t, err == nil)
		defer origDstF.Close()
		
		dstF, err := os.Create(dstPath)
		assert.T(t, err == nil)
		defer dstF.Close()
		
		io.Copy(dstF, origDstF)
	}()
	
	err := PatchFile(srcPath, dstPath)
	if err != nil { fmt.Print(err.String()) }
	assert.T(t, err == nil)
	
	srcFile, err := blocks.IndexFile(srcPath)
	assert.T(t, err == nil)
	
	dstFile, err := blocks.IndexFile(dstPath)
	assert.T(t, err == nil)
	
	assert.Equal(t, srcFile.Strong(), dstFile.Strong())
}

// Test the patch planner on two identical directory structures.
func TestPatchIdentity(t *testing.T) {
	tg := treegen.New()
	treeSpec := tg.D("foo", tg.F("bar", tg.B(42, 65537)))
	
	srcpath := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(srcpath)
	srcStore, err := blocks.NewLocalStore(srcpath)
	assert.T(t, err == nil)
	
	dstpath := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(dstpath)
	dstStore, err := blocks.NewLocalStore(dstpath)
	assert.T(t, err == nil)
	
	patchPlan := NewPatchPlan(srcStore, dstStore)
//	printPlan(patchPlan)
	
	assert.T(t, len(patchPlan.Cmds) > 0)
	for i := 0; i < len(patchPlan.Cmds); i++ {
		keep := patchPlan.Cmds[0].(*Keep)
		assert.T(t, strings.HasPrefix(dstpath, keep.Path.Resolve()))
	}
}

// Test the matcher on a case where the source file has the same 
// prefix as destination, but has been appended to.
func TestMatchAppend(t *testing.T) {
	tg := treegen.New()
	treeSpec := tg.F("bar", tg.B(42, 65537), tg.B(43, 65537))
	
	srcpath := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(srcpath)
	
	// Try indexing root dir as a file
	srcFile, err := blocks.IndexFile(srcpath)
	assert.Tf(t, err != nil, "%v", err)
	
	// Ok, for real this time
	srcFile, err = blocks.IndexFile(filepath.Join(srcpath, "bar"))
	assert.Tf(t, err == nil, "%v", err)
	assert.Equal(t, 17, len(srcFile.Blocks))
	
	tg = treegen.New()
	treeSpec = tg.F("bar", tg.B(42, 65537))
	
	dstpath := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(dstpath)
	dstFile, err := blocks.IndexFile(filepath.Join(dstpath, "bar"))
	assert.Equal(t, 9, len(dstFile.Blocks))
	
	match, err := MatchFile(srcFile, filepath.Join(dstpath, "bar"))
	assert.T(t, err == nil, "%v", err)
	
	assert.Equal(t, 8, len(match.BlockMatches))
	
	notMatched := match.NotMatched()
	assert.Equal(t, 1, len(notMatched))
	assert.Equal(t, int64(65536), notMatched[0].From)
	assert.Equal(t, int64(65537+65537), notMatched[0].To)
}

// Test the patch planner on a case where the source file has the same 
// prefix as destination, but has been appended to.
// Execute the patch plan and check both resulting trees are identical.
func TestPatchFileAppend(t *testing.T) {
	tg := treegen.New()
	treeSpec := tg.D("foo", tg.F("bar", tg.B(42, 65537), tg.B(43, 65537)))
	
	srcpath := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(srcpath)
	srcStore, err := blocks.NewLocalStore(srcpath)
	assert.T(t, err == nil)
	
	tg = treegen.New()
	treeSpec = tg.D("foo", tg.F("bar", tg.B(42, 65537)))
	
	dstpath := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(dstpath)
	dstStore, err := blocks.NewLocalStore(dstpath)
	assert.T(t, err == nil)
	
	patchPlan := NewPatchPlan(srcStore, dstStore)
//	printPlan(patchPlan)
	
	complete := false
	for i, cmd := range patchPlan.Cmds {
		switch {
		case i == 0:
			localTemp, isTemp := cmd.(*LocalTemp)
			assert.T(t, isTemp)
			assert.Equal(t, filepath.Join(dstpath, "foo", "bar"), localTemp.Path.Resolve())
		case i >= 1 && i <=8:
			ltc, isLtc := cmd.(*LocalTempCopy)
			assert.Tf(t, isLtc, "cmd %d", i)
			assert.Equal(t, ltc.LocalOffset, ltc.TempOffset)
			assert.Equal(t, int64(blocks.BLOCKSIZE), ltc.Length)
			assert.Equal(t, int64(0), ltc.LocalOffset % int64(blocks.BLOCKSIZE))
		case i == 9:
			stc, isStc := cmd.(*SrcTempCopy)
			assert.T(t, isStc)
			assert.Equal(t, int64(65538), stc.Length)
		case i == 10:
			_, isRwt := cmd.(*ReplaceWithTemp)
			assert.T(t, isRwt)
			complete = true
		case i > 10:
			t.Fatalf("too many commands")
		}
	}
	assert.T(t, complete, "missing expected number of commands")
	
	failedCmd, err := patchPlan.Exec()
	assert.Tf(t, failedCmd == nil && err == nil, "%v: %v", failedCmd, err)
	
	srcRoot, _ := blocks.IndexDir(srcpath)
	dstRoot, _ := blocks.IndexDir(dstpath)
	assert.Equal(t, srcRoot.Strong(), dstRoot.Strong())
}

// Test the patch planner on a case where the source file is a shorter,
// truncated version of the destination.
// Execute the patch plan and check both resulting trees are identical.
func TestPatchFileTruncate(t *testing.T) {
	tg := treegen.New()
	treeSpec := tg.D("foo", tg.F("bar", tg.B(42, 65537)))
	
	srcpath := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(srcpath)
	srcStore, err := blocks.NewLocalStore(srcpath)
	assert.T(t, err == nil)
	
	tg = treegen.New()
	treeSpec = tg.D("foo", tg.F("bar", tg.B(42, 65537), tg.B(43, 65537)))
	
	dstpath := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(dstpath)
	dstStore, err := blocks.NewLocalStore(dstpath)
	assert.T(t, err == nil)
	
	patchPlan := NewPatchPlan(srcStore, dstStore)
//	printPlan(patchPlan)
	
	complete := false
	for i, cmd := range patchPlan.Cmds {
		switch {
		case i == 0:
			localTemp, isTemp := cmd.(*LocalTemp)
			assert.T(t, isTemp)
			assert.Equal(t, filepath.Join(dstpath, "foo", "bar"), localTemp.Path.Resolve())
		case i >= 1 && i <=8:
			ltc, isLtc := cmd.(*LocalTempCopy)
			assert.Tf(t, isLtc, "cmd %d", i)
			assert.Equal(t, ltc.LocalOffset, ltc.TempOffset)
			assert.Equal(t, int64(blocks.BLOCKSIZE), ltc.Length)
			assert.Equal(t, int64(0), ltc.LocalOffset % int64(blocks.BLOCKSIZE))
		case i == 9:
			stc, isStc := cmd.(*SrcTempCopy)
			assert.T(t, isStc)
			assert.Equal(t, int64(1), stc.Length)
			complete = true
		case i > 10:
			t.Fatalf("too many commands")
		}
	}
	assert.T(t, complete, "missing expected number of commands")
	
	failedCmd, err := patchPlan.Exec()
	assert.Tf(t, failedCmd == nil && err == nil, "%v: %v", failedCmd, err)
	
	srcRoot, _ := blocks.IndexDir(srcpath)
	dstRoot, _ := blocks.IndexDir(dstpath)
	assert.Equal(t, srcRoot.Strong(), dstRoot.Strong())
}

// Test the patch planner's ability to track adding a bunch of new files.
func TestPatchAdd(t *testing.T) {
	tg := treegen.New()
	
	files := []treegen.Generated{}
	for i := 0; i < 10; i++ {
		files = append(files, tg.F("", tg.B(int64(42*i), int64(500000*i))))
	}
	
	treeSpec := tg.D("foo", tg.D("bar", files...))
	srcpath := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(srcpath)
	srcStore, err := blocks.NewLocalStore(filepath.Join(srcpath, "foo"))
	assert.T(t, err == nil)
	
	tg = treegen.New()
	treeSpec = tg.D("foo", tg.D("bar"), tg.D("baz"))
	dstpath := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(dstpath)
	dstStore, err := blocks.NewLocalStore(filepath.Join(dstpath, "foo"))
	assert.T(t, err == nil)
	
	patchPlan := NewPatchPlan(srcStore, dstStore)
	
//	printPlan(patchPlan)
	
	for _, cmd := range patchPlan.Cmds {
		_, isSfd := cmd.(*SrcFileDownload)
		assert.T(t, isSfd)
	}
}

// Test patch planner on a file rename. Contents remain the same.
func TestPatchRenameFileSameDir(t *testing.T) {
	tg := treegen.New()
	treeSpec := tg.D("foo", tg.F("bar", tg.B(42, 65537)))
	
	srcpath := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(srcpath)
	srcStore, err := blocks.NewLocalStore(srcpath)
	assert.T(t, err == nil)
	
	tg = treegen.New()
	treeSpec = tg.D("foo", tg.F("baz", tg.B(42, 65537)))
	
	dstpath := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(dstpath)
	dstStore, err := blocks.NewLocalStore(dstpath)
	assert.T(t, err == nil)
	
	patchPlan := NewPatchPlan(srcStore, dstStore)
	
	assert.Equal(t, 1, len(patchPlan.Cmds))
	rename, isRename := patchPlan.Cmds[0].(*Rename)
	assert.T(t, isRename)
	assert.T(t, strings.HasSuffix(rename.From.Resolve(), filepath.Join("foo", "baz")))
	assert.T(t, strings.HasSuffix(rename.To.Resolve(), filepath.Join("foo", "bar")))
}

// Test patch planner on a file directory restructuring between 
// source and destination, where files have identical content in both.
func TestPatchRenameFileDifferentDir(t *testing.T) {
	tg := treegen.New()
	treeSpec := tg.D("foo", 
					tg.D("gloo", 
						tg.F("bloo", tg.B(99, 99)), 
						tg.D("groo", 
							tg.D("snoo", 
								tg.F("bar", tg.B(42, 65537))))))
	
	srcpath := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(srcpath)
	srcStore, err := blocks.NewLocalStore(srcpath)
	assert.T(t, err == nil)
	
	tg = treegen.New()
	treeSpec = tg.D("pancake", 
					tg.F("butter", tg.B(42, 65537)), 
					tg.F("syrup", tg.B(99, 99)))
	
	dstpath := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(dstpath)
	dstStore, err := blocks.NewLocalStore(dstpath)
	assert.T(t, err == nil)
	
	patchPlan := NewPatchPlan(srcStore, dstStore)
	assert.Equal(t, 2, len(patchPlan.Cmds))
	for i := 0; i < len(patchPlan.Cmds); i++ {
		_, isRename := patchPlan.Cmds[0].(*Rename)
		assert.T(t, isRename)
	}
	
	// Now flip
	patchPlan = NewPatchPlan(dstStore, srcStore)
	assert.Equal(t, 2, len(patchPlan.Cmds))
	for i := 0; i < len(patchPlan.Cmds); i++ {
		_, isRename := patchPlan.Cmds[0].(*Rename)
		assert.T(t, isRename)
	}
}

// Test patch planner on case where the source and 
// destination have a direct conflict in structure.
// A path in the source is a directory, path in destination 
// already contains a file at that location.
func TestPatchSimpleDirFileConflict(t *testing.T) {
	tg := treegen.New()
	treeSpec := tg.D("foo", 
					tg.D("gloo", 
						tg.F("bloo", tg.B(99, 99)), 
						tg.D("groo", 
							tg.D("snoo", 
								tg.F("bar", tg.B(42, 65537))))))
	
	srcpath := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(srcpath)
	srcStore, err := blocks.NewLocalStore(srcpath)
	assert.T(t, err == nil)
	
	tg = treegen.New()
	treeSpec = tg.D("foo", 
					tg.F("gloo", tg.B(99, 999)))
	
	dstpath := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(dstpath)
	dstStore, err := blocks.NewLocalStore(dstpath)
	assert.T(t, err == nil)
	
	patchPlan := NewPatchPlan(srcStore, dstStore)
//	printPlan(patchPlan)
	
	failedCmd, err := patchPlan.Exec()
	assert.Tf(t, failedCmd == nil && err == nil, "%v: %v", failedCmd, err)
	
	assert.Equal(t, 3, len(patchPlan.Cmds))
	for i, cmd := range patchPlan.Cmds {
		switch i {
		case 0:
			conflict, is := cmd.(*Conflict)
			assert.T(t, is)
			assert.T(t, strings.HasSuffix(conflict.Path.RelPath, "foo/gloo"))
		case 1:
			copy, is := cmd.(*SrcFileDownload)
			assert.T(t, is)
			assert.Equal(t, "beced72da0cf22301e23bdccec61bf9763effd6f", copy.SrcFile.Strong())
		case 2:
			copy, is := cmd.(*SrcFileDownload)
			assert.T(t, is)
			assert.Equal(t, "764b5f659f70e69d4a87fe6ed138af40be36c514", copy.SrcFile.Strong())
		}
	}
}

func assertNoRelocs(t *testing.T, path string) {
	d, err := os.Open(path)
	assert.T(t, err == nil)
	
	names, err := d.Readdirnames(0)
	assert.T(t, err == nil)
	
	for _, name := range names {
		assert.T(t, !strings.HasPrefix(name, "_reloc"))
	}
}

// Test patch planner on case where the source and 
// destination have a direct conflict in structure.
// A path in the source is a directory, path in destination 
// already contains a file at that location.
func TestPatchRelocConflict(t *testing.T) {
	tg := treegen.New()
	treeSpec := tg.D("foo", 
					tg.D("gloo", 
						tg.F("bloo", tg.B(99, 99)), 
						tg.D("groo", 
							tg.D("snoo", 
								tg.F("bar", tg.B(42, 65537))))))
	
	srcpath := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(srcpath)
	srcStore, err := blocks.NewLocalStore(srcpath)
	assert.T(t, err == nil)
	
	tg = treegen.New()
	treeSpec = tg.D("foo", 
					tg.F("gloo", tg.B(99, 99)))
	
	dstpath := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(dstpath)
	dstStore, err := blocks.NewLocalStore(dstpath)
	assert.T(t, err == nil)
	
	patchPlan := NewPatchPlan(srcStore, dstStore)
//	printPlan(patchPlan)
	
	assert.Equal(t, 3, len(patchPlan.Cmds))
	for i, cmd := range patchPlan.Cmds {
		switch i {
		case 0:
			conflict, is := cmd.(*Conflict)
			assert.T(t, is)
			assert.T(t, strings.HasSuffix(conflict.Path.RelPath, "foo/gloo"))
		case 1:
			rename, is := cmd.(*Rename)
			assert.T(t, is)
			assert.T(t, strings.HasSuffix(rename.From.Resolve(), "foo/gloo"))
			assert.T(t, strings.HasSuffix(rename.To.Resolve(), "foo/gloo/bloo"))
		case 2:
			copy, is := cmd.(*SrcFileDownload)
			assert.T(t, is)
			assert.Equal(t, "764b5f659f70e69d4a87fe6ed138af40be36c514", copy.SrcFile.Strong())
		}
	}
	
	failedCmd, err := patchPlan.Exec()
	assert.Tf(t, failedCmd == nil && err == nil, "%v: %v", failedCmd, err)
	
	assertNoRelocs(t, dstpath)
}


func TestPatchDepConflict(t *testing.T) {
	tg := treegen.New()
	treeSpec := tg.D("foo", 
					tg.D("gloo", 
						tg.F("bloo", tg.B(99, 8192), tg.B(100, 10000)), 
						tg.D("groo", 
							tg.D("snoo", 
								tg.F("bar", tg.B(42, 65537))))))
	
	srcpath := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(srcpath)
	srcStore, err := blocks.NewLocalStore(srcpath)
	assert.T(t, err == nil)
	
	tg = treegen.New()
	treeSpec = tg.D("foo", 
					tg.F("gloo", tg.B(99, 10000)))
	
	dstpath := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(dstpath)
	dstStore, err := blocks.NewLocalStore(dstpath)
	assert.T(t, err == nil)
	
	patchPlan := NewPatchPlan(srcStore, dstStore)
//	printPlan(patchPlan)
	
	failedCmd, err := patchPlan.Exec()
	assert.Tf(t, failedCmd == nil && err == nil, "%v: %v", failedCmd, err)
	
	assertNoRelocs(t, dstpath)
}

func TestPatchWeakCollision(t *testing.T) {
	tg := treegen.New()
	treeSpec := tg.D("foo", 
					tg.F("bar", tg.B(6806, 65536)))
	
	srcpath := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(srcpath)
	srcStore, err := blocks.NewLocalStore(srcpath)
	assert.T(t, err == nil)
	
	tg = treegen.New()
	treeSpec = tg.D("foo", 
					tg.F("bar", tg.B(9869, 65536)))
	
	dstpath := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(dstpath)
	dstStore, err := blocks.NewLocalStore(dstpath)
	assert.T(t, err == nil)
	
	// Src and dst blocks have same weak checksum
	assert.Equal(t,
		srcStore.Root().SubDirs[0].Files[0].Blocks[0].Weak(),
		dstStore.Root().SubDirs[0].Files[0].Blocks[0].Weak())
	
	// Src and dst blocks have different strong checksum
	assert.Tf(t, srcStore.Root().Strong() != dstStore.Root().Strong(),
		"wtf: %v == %v", srcStore.Root().Strong(), dstStore.Root().Strong())
	
	patchPlan := NewPatchPlan(srcStore, dstStore)
//	printPlan(patchPlan)
	
	failedCmd, err := patchPlan.Exec()
	assert.Tf(t, failedCmd == nil && err == nil, "%v: %v", failedCmd, err)
	
	srcDir, err := blocks.IndexDir(srcpath)
	assert.T(t, err == nil)
	dstDir, err := blocks.IndexDir(dstpath)
	assert.T(t, err == nil)
	
	assert.Equal(t, srcDir.Strong(), dstDir.Strong())
}



