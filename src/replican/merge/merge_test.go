
package merge

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"replican/blocks"
	"replican/treegen"
	"testing"
	
	"github.com/bmizerany/assert"
)

func printPlan(plan *PatchPlan) {
	for i := 0; i < len(plan.Cmds); i++ {
		fmt.Printf("%s\n", plan.Cmds[i].String())
	}
}

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
	
	assert.Equal(t, 1, len(patchPlan.Cmds))
	keep := patchPlan.Cmds[0].(*Keep)
	assert.Equal(t, dstpath, keep.Path)
}

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
			assert.Equal(t, filepath.Join(dstpath, "foo", "bar"), localTemp.Path)
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
			assert.Equal(t, filepath.Join(dstpath, "foo", "bar"), localTemp.Path)
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

func TestPatchAdd(t *testing.T) {
	tg := treegen.New()
	
	files := []treegen.Generated{}
	for i := 0; i < 10; i++ {
		files = append(files, tg.F("", tg.B(int64(42*i), int64(500000*i))))
	}
	
	treeSpec := tg.D("foo", tg.D("bar", files...))
	srcpath := treegen.TestTree(t, treeSpec)
	srcStore, err := blocks.NewLocalStore(filepath.Join(srcpath, "foo"))
	assert.T(t, err == nil)
	
	tg = treegen.New()
	treeSpec = tg.D("foo", tg.D("bar"), tg.D("baz"))
	dstpath := treegen.TestTree(t, treeSpec)
	dstStore, err := blocks.NewLocalStore(filepath.Join(dstpath, "foo"))
	assert.T(t, err == nil)
	
	patchPlan := NewPatchPlan(srcStore, dstStore)
	
	printPlan(patchPlan)
	
	for _, cmd := range patchPlan.Cmds {
		_, isSfd := cmd.(*SrcFileDownload)
		assert.T(t, isSfd)
	}
}



