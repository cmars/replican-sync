package sync

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"github.com/cmars/replican-sync/replican/fs"
	"github.com/cmars/replican-sync/replican/fs/sqlite3"
	"github.com/cmars/replican-sync/replican/treegen"
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

// Test an actual file patch on the munged file scenario from TestMatchMunge.
// Resulting patched file should be identical to the source file.
func TestPatch(t *testing.T) {
	srcPath := "../../testroot/My Music/0 10k 30.mp4"
	dstPath := filepath.Join(os.TempDir(), "foo.mp4")
	os.RemoveAll(dstPath)

	origDstF, err := os.Open("../../testroot/My Music/0 10k 30 munged.mp4")
	assert.Tf(t, err == nil, "%v", err)

	dstF, err := os.Create(dstPath)
	assert.Tf(t, err == nil, "%v", err)

	_, err = io.Copy(dstF, origDstF)
	assert.Tf(t, err == nil, "%v", err)

	origDstF.Close()
	dstF.Close()

	patchPlan, err := Patch(srcPath, dstPath)
	//	printPlan(patchPlan)

	assert.Tf(t, err == nil, "%v", err)
	failedCmd, err := patchPlan.Exec()
	assert.Tf(t, failedCmd == nil && err == nil, "%v: %v", failedCmd, err)

	srcFileInfo, _, err := fs.IndexFile(srcPath)
	assert.T(t, err == nil)

	dstFileInfo, _, err := fs.IndexFile(dstPath)
	assert.Tf(t, err == nil, "%v", err)

	assert.Equal(t, srcFileInfo.Strong, dstFileInfo.Strong)
}

type repoMaker func(t *testing.T) fs.NodeRepo

func mkMemRepo(_ *testing.T) fs.NodeRepo {
	return fs.NewMemRepo()
}

func mkDbRepo(t *testing.T) fs.NodeRepo {
	dbrepo, err := sqlite3.NewDbRepo(":memory:")
	assert.T(t, err == nil)
	return dbrepo
}

// Test the patch planner on two identical directory structures.

func TestPatchIdentity(t *testing.T) {
	DoTestPatchIdentity(t, mkMemRepo)
}

func TestDbPatchIdentity(t *testing.T) {
	DoTestPatchIdentity(t, mkDbRepo)
}

func DoTestPatchIdentity(t *testing.T, mkrepo repoMaker) {
	tg := treegen.New()
	treeSpec := tg.D("foo", tg.F("bar", tg.B(42, 65537)))

	srcpath := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(srcpath)
	srcRepo := mkrepo(t)
	defer srcRepo.Close()
	srcStore, err := fs.NewLocalStore(srcpath, srcRepo)
	assert.T(t, err == nil)
	
	assert.Equal(t, "1b1979b8746948cedc81488e92d1ad715e38bbfc", 
		srcStore.Repo().Root().(fs.Dir).Info().Strong)
	
	dstpath := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(dstpath)
	dstRepo := mkrepo(t)
	defer dstRepo.Close()
	dstStore, err := fs.NewLocalStore(dstpath, dstRepo)
	assert.T(t, err == nil)

	assert.Equal(t, "1b1979b8746948cedc81488e92d1ad715e38bbfc", 
		dstStore.Repo().Root().(fs.Dir).Info().Strong)
	
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
	srcFileInfo, srcBlocksInfo, err := fs.IndexFile(srcpath)
	assert.Tf(t, err != nil, "%v", err)

	// Ok, for real this time
	srcFileInfo, srcBlocksInfo, err = fs.IndexFile(filepath.Join(srcpath, "bar"))
	assert.Tf(t, err == nil, "%v", err)
	assert.Equal(t, 17, len(srcBlocksInfo))

	tg = treegen.New()
	treeSpec = tg.F("bar", tg.B(42, 65537))

	dstpath := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(dstpath)
	_, dstBlocksInfo, err := fs.IndexFile(filepath.Join(dstpath, "bar"))
	assert.Equal(t, 9, len(dstBlocksInfo))

	srcRepo := fs.NewMemRepo()
	srcFile := srcRepo.AddFile(nil, srcFileInfo, srcBlocksInfo)

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
	DoTestPatchFileAppend(t, mkMemRepo)
}

func TestDbPatchFileAppend(t *testing.T) {
	DoTestPatchFileAppend(t, mkDbRepo)
}

func DoTestPatchFileAppend(t *testing.T, mkrepo repoMaker) {
	tg := treegen.New()
	treeSpec := tg.D("foo", tg.F("bar", tg.B(42, 65537), tg.B(43, 65537)))

	srcpath := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(srcpath)
	srcRepo := mkrepo(t)
	defer srcRepo.Close()
	srcStore, err := fs.NewLocalStore(srcpath, srcRepo)
	assert.T(t, err == nil)

	tg = treegen.New()
	treeSpec = tg.D("foo", tg.F("bar", tg.B(42, 65537)))

	dstpath := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(dstpath)
	dstRepo := mkrepo(t)
	defer dstRepo.Close()
	dstStore, err := fs.NewLocalStore(dstpath, dstRepo)
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
		case i >= 1 && i <= 8:
			ltc, isLtc := cmd.(*LocalTempCopy)
			assert.Tf(t, isLtc, "cmd %d", i)
			assert.Equal(t, ltc.LocalOffset, ltc.TempOffset)
			assert.Equal(t, int64(fs.BLOCKSIZE), ltc.Length)
			assert.Equal(t, int64(0), ltc.LocalOffset%int64(fs.BLOCKSIZE))
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

	errorChan := make(chan os.Error)
	go func() {
		srcRoot := fs.IndexDir(srcpath, fs.NewMemRepo(), errorChan)
		dstRoot := fs.IndexDir(dstpath, fs.NewMemRepo(), errorChan)
		assert.Equal(t, srcRoot.Info().Strong, dstRoot.Info().Strong)
		close(errorChan)
	}()
	for err := range errorChan {
		assert.Tf(t, err == nil, "%v", err)
	}
}

// Test the patch planner on a case where the source file is a shorter,
// truncated version of the destination.
// Execute the patch plan and check both resulting trees are identical.

func TestPatchFileTruncate(t *testing.T) {
	DoTestPatchFileTruncate(t, mkMemRepo)
}

func TestDbPatchFileTruncate(t *testing.T) {
	DoTestPatchFileTruncate(t, mkDbRepo)
}

func DoTestPatchFileTruncate(t *testing.T, mkrepo repoMaker) {
	tg := treegen.New()
	treeSpec := tg.D("foo", tg.F("bar", tg.B(42, 65537)))

	srcpath := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(srcpath)
	srcRepo := mkrepo(t)
	defer srcRepo.Close()
	srcStore, err := fs.NewLocalStore(srcpath, srcRepo)
	assert.T(t, err == nil)

	tg = treegen.New()
	treeSpec = tg.D("foo", tg.F("bar", tg.B(42, 65537), tg.B(43, 65537)))

	dstpath := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(dstpath)
	dstRepo := mkrepo(t)
	defer dstRepo.Close()
	dstStore, err := fs.NewLocalStore(dstpath, dstRepo)
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
		case i >= 1 && i <= 8:
			ltc, isLtc := cmd.(*LocalTempCopy)
			assert.Tf(t, isLtc, "cmd %d", i)
			assert.Equal(t, ltc.LocalOffset, ltc.TempOffset)
			assert.Equal(t, int64(fs.BLOCKSIZE), ltc.Length)
			assert.Equal(t, int64(0), ltc.LocalOffset%int64(fs.BLOCKSIZE))
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

	errorChan := make(chan os.Error)
	go func() {
		srcRoot := fs.IndexDir(srcpath, fs.NewMemRepo(), errorChan)
		dstRoot := fs.IndexDir(dstpath, fs.NewMemRepo(), errorChan)
		assert.Equal(t, srcRoot.Info().Strong, dstRoot.Info().Strong)
		close(errorChan)
	}()
	for err := range errorChan {
		assert.Tf(t, err == nil, "%v", err)
	}
}

// Test the patch planner's ability to track adding a bunch of new files.

func TestPatchAdd(t *testing.T) {
	DoTestPatchAdd(t, mkMemRepo)
}

func TestDbPatchAdd(t *testing.T) {
	DoTestPatchAdd(t, mkDbRepo)
}

func DoTestPatchAdd(t *testing.T, mkrepo repoMaker) {
	tg := treegen.New()

	files := []treegen.Generated{}
	for i := 0; i < 10; i++ {
		files = append(files, tg.F("", tg.B(int64(42*i), int64(500000*i))))
	}

	treeSpec := tg.D("foo", tg.D("bar", files...))
	srcpath := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(srcpath)
	srcRepo := mkrepo(t)
	defer srcRepo.Close()
	srcStore, err := fs.NewLocalStore(filepath.Join(srcpath, "foo"), srcRepo)
	assert.T(t, err == nil)

	tg = treegen.New()
	treeSpec = tg.D("foo", tg.D("bar"), tg.D("baz"))
	dstpath := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(dstpath)
	dstRepo := mkrepo(t)
	defer dstRepo.Close()
	dstStore, err := fs.NewLocalStore(filepath.Join(dstpath, "foo"), dstRepo)
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
	DoTestPatchRenameFileSameDir(t, mkMemRepo)
}

func TestDbPatchRenameFileSameDir(t *testing.T) {
	DoTestPatchRenameFileSameDir(t, mkDbRepo)
}

func DoTestPatchRenameFileSameDir(t *testing.T, mkrepo repoMaker) {
	tg := treegen.New()
	treeSpec := tg.D("foo", tg.F("bar", tg.B(42, 65537)))

	srcpath := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(srcpath)
	srcRepo := mkrepo(t)
	defer srcRepo.Close()
	srcStore, err := fs.NewLocalStore(srcpath, srcRepo)
	assert.T(t, err == nil)

	tg = treegen.New()
	treeSpec = tg.D("foo", tg.F("baz", tg.B(42, 65537)))

	dstpath := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(dstpath)
	dstRepo := mkrepo(t)
	defer dstRepo.Close()
	dstStore, err := fs.NewLocalStore(dstpath, dstRepo)
	assert.T(t, err == nil)

	patchPlan := NewPatchPlan(srcStore, dstStore)

	assert.Equal(t, 1, len(patchPlan.Cmds))
	rename, isRename := patchPlan.Cmds[0].(*Transfer)
	assert.T(t, isRename)
	assert.T(t, strings.HasSuffix(rename.From.Resolve(), filepath.Join("foo", "baz")))
	assert.T(t, strings.HasSuffix(rename.To.Resolve(), filepath.Join("foo", "bar")))
}

// Test patch planner on a file directory restructuring between 
// source and destination, where files have identical content in both.

func TestPatchRenameFileDifferentDir(t *testing.T) {
	DoTestPatchRenameFileDifferentDir(t, mkMemRepo)
}

func TestDbPatchRenameFileDifferentDir(t *testing.T) {
	DoTestPatchRenameFileDifferentDir(t, mkDbRepo)
}

func DoTestPatchRenameFileDifferentDir(t *testing.T, mkrepo repoMaker) {
	tg := treegen.New()
	treeSpec := tg.D("foo",
		tg.D("gloo",
			tg.F("bloo", tg.B(99, 99)),
			tg.D("groo",
				tg.D("snoo",
					tg.F("bar", tg.B(42, 65537))))))

	srcpath := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(srcpath)
	srcRepo := mkrepo(t)
	defer srcRepo.Close()
	srcStore, err := fs.NewLocalStore(srcpath, srcRepo)
	assert.T(t, err == nil)

	tg = treegen.New()
	treeSpec = tg.D("pancake",
		tg.F("butter", tg.B(42, 65537)),
		tg.F("syrup", tg.B(99, 99)))

	dstpath := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(dstpath)
	dstRepo := mkrepo(t)
	defer dstRepo.Close()
	dstStore, err := fs.NewLocalStore(dstpath, dstRepo)
	assert.T(t, err == nil)

	patchPlan := NewPatchPlan(srcStore, dstStore)
	assert.Equal(t, 2, len(patchPlan.Cmds))
	for i := 0; i < len(patchPlan.Cmds); i++ {
		_, isRename := patchPlan.Cmds[0].(*Transfer)
		assert.T(t, isRename)
	}

	// Now flip
	patchPlan = NewPatchPlan(dstStore, srcStore)
	assert.Equal(t, 2, len(patchPlan.Cmds))
	for i := 0; i < len(patchPlan.Cmds); i++ {
		_, isRename := patchPlan.Cmds[0].(*Transfer)
		assert.T(t, isRename)
	}
}

// Test patch planner on case where the source and 
// destination have a direct conflict in structure.
// A path in the source is a directory, path in destination 
// already contains a file at that location.

func TestPatchSimpleDirFileConflict(t *testing.T) {
	DoTestPatchSimpleDirFileConflict(t, mkMemRepo)
}

func TestDbPatchSimpleDirFileConflict(t *testing.T) {
	DoTestPatchSimpleDirFileConflict(t, mkDbRepo)
}

func DoTestPatchSimpleDirFileConflict(t *testing.T, mkrepo repoMaker) {
	tg := treegen.New()
	treeSpec := tg.D("foo",
		tg.D("gloo",
			tg.F("bloo", tg.B(99, 99)),
			tg.D("groo",
				tg.D("snoo",
					tg.F("bar", tg.B(42, 65537))))))

	srcpath := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(srcpath)
	srcRepo := mkrepo(t)
	defer srcRepo.Close()
	srcStore, err := fs.NewLocalStore(srcpath, srcRepo)
	assert.T(t, err == nil)

	tg = treegen.New()
	treeSpec = tg.D("foo",
		tg.F("gloo", tg.B(99, 999)))

	dstpath := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(dstpath)
	dstRepo := mkrepo(t)
	defer dstRepo.Close()
	dstStore, err := fs.NewLocalStore(dstpath, dstRepo)
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
			assert.T(t, strings.HasSuffix(conflict.Path.RelPath, filepath.Join("foo", "gloo")))
		case 1:
			copy, is := cmd.(*SrcFileDownload)
			assert.T(t, is)
			assert.Equal(t, "beced72da0cf22301e23bdccec61bf9763effd6f", copy.SrcFile.Info().Strong)
		case 2:
			copy, is := cmd.(*SrcFileDownload)
			assert.T(t, is)
			assert.Equal(t, "764b5f659f70e69d4a87fe6ed138af40be36c514", copy.SrcFile.Info().Strong)
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
	DoTestPatchRelocConflict(t, mkMemRepo)
}

func TestDbPatchRelocConflict(t *testing.T) {
	DoTestPatchRelocConflict(t, mkDbRepo)
}

func DoTestPatchRelocConflict(t *testing.T, mkrepo repoMaker) {
	tg := treegen.New()
	treeSpec := tg.D("foo",
		tg.D("gloo",
			tg.F("bloo", tg.B(99, 99)),
			tg.D("groo",
				tg.D("snoo",
					tg.F("bar", tg.B(42, 65537))))))

	srcpath := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(srcpath)
	srcRepo := mkrepo(t)
	defer srcRepo.Close()
	srcStore, err := fs.NewLocalStore(srcpath, srcRepo)
	assert.T(t, err == nil)

	tg = treegen.New()
	treeSpec = tg.D("foo",
		tg.F("gloo", tg.B(99, 99)))

	dstpath := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(dstpath)
	dstRepo := mkrepo(t)
	defer dstRepo.Close()
	dstStore, err := fs.NewLocalStore(dstpath, dstRepo)
	assert.T(t, err == nil)

	patchPlan := NewPatchPlan(srcStore, dstStore)
	//	printPlan(patchPlan)

	assert.Equal(t, 3, len(patchPlan.Cmds))
	for i, cmd := range patchPlan.Cmds {
		switch i {
		case 0:
			conflict, is := cmd.(*Conflict)
			assert.T(t, is)
			assert.T(t, strings.HasSuffix(conflict.Path.RelPath, filepath.Join("foo", "gloo")))
		case 1:
			copy, is := cmd.(*Transfer)
			assert.T(t, is)
			assert.T(t, strings.HasSuffix(copy.From.Resolve(), filepath.Join("foo", "gloo")))
			assert.T(t, strings.HasSuffix(copy.To.Resolve(), filepath.Join("foo", "gloo", "bloo")))
		case 2:
			copy, is := cmd.(*SrcFileDownload)
			assert.T(t, is)
			assert.Equal(t, "764b5f659f70e69d4a87fe6ed138af40be36c514", copy.SrcFile.Info().Strong)
		}
	}

	failedCmd, err := patchPlan.Exec()
	assert.Tf(t, failedCmd == nil && err == nil, "%v: %v", failedCmd, err)

	assertNoRelocs(t, dstpath)
}

func TestPatchDepConflict(t *testing.T) {
	DoTestPatchDepConflict(t, mkMemRepo)
}

func TestDbPatchDepConflict(t *testing.T) {
	DoTestPatchDepConflict(t, mkDbRepo)
}

func DoTestPatchDepConflict(t *testing.T, mkrepo repoMaker) {
	tg := treegen.New()
	treeSpec := tg.D("foo",
		tg.D("gloo",
			tg.F("bloo", tg.B(99, 8192), tg.B(100, 10000)),
			tg.D("groo",
				tg.D("snoo",
					tg.F("bar", tg.B(42, 65537))))))

	srcpath := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(srcpath)
	srcRepo := mkrepo(t)
	defer srcRepo.Close()
	srcStore, err := fs.NewLocalStore(srcpath, srcRepo)
	assert.T(t, err == nil)

	tg = treegen.New()
	treeSpec = tg.D("foo",
		tg.F("gloo", tg.B(99, 10000)))

	dstpath := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(dstpath)
	dstRepo := mkrepo(t)
	defer dstRepo.Close()
	dstStore, err := fs.NewLocalStore(dstpath, dstRepo)
	assert.T(t, err == nil)

	patchPlan := NewPatchPlan(srcStore, dstStore)
	//	printPlan(patchPlan)

	failedCmd, err := patchPlan.Exec()
	assert.Tf(t, failedCmd == nil && err == nil, "%v: %v", failedCmd, err)

	assertNoRelocs(t, dstpath)
}

func TestPatchWeakCollision(t *testing.T) {
	DoTestPatchWeakCollision(t, mkMemRepo)
}

func TestDbPatchWeakCollision(t *testing.T) {
	DoTestPatchWeakCollision(t, mkDbRepo)
}

func DoTestPatchWeakCollision(t *testing.T, mkrepo repoMaker) {
	tg := treegen.New()
	treeSpec := tg.D("foo",
		tg.F("bar", tg.B(6806, 65536)))

	srcpath := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(srcpath)
	srcRepo := mkrepo(t)
	defer srcRepo.Close()
	srcStore, err := fs.NewLocalStore(srcpath, srcRepo)
	assert.T(t, err == nil)

	tg = treegen.New()
	treeSpec = tg.D("foo",
		tg.F("bar", tg.B(9869, 65536)))

	dstpath := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(dstpath)
	dstRepo := mkrepo(t)
	defer dstRepo.Close()
	dstStore, err := fs.NewLocalStore(dstpath, dstRepo)
	assert.T(t, err == nil)

	// Src and dst blocks have same weak checksum
	assert.Equal(t,
		(srcStore.Repo().Root().(fs.Dir)).SubDirs()[0].Files()[0].Blocks()[0].Info().Weak,
		(dstStore.Repo().Root().(fs.Dir)).SubDirs()[0].Files()[0].Blocks()[0].Info().Weak)

	// Src and dst blocks have different strong checksum
	srcRoot := srcStore.Repo().Root().(fs.Dir)
	dstRoot := dstStore.Repo().Root().(fs.Dir)
	assert.Tf(t, srcRoot.Info().Strong != dstRoot.Info().Strong,
		"wtf: %v == %v", srcRoot.Info().Strong, dstRoot.Info().Strong)

	patchPlan := NewPatchPlan(srcStore, dstStore)
	//	printPlan(patchPlan)

	failedCmd, err := patchPlan.Exec()
	assert.Tf(t, failedCmd == nil && err == nil, "%v: %v", failedCmd, err)

	errorChan := make(chan os.Error)
	go func() {
		srcDir := fs.IndexDir(srcpath, fs.NewMemRepo(), errorChan)
		dstDir := fs.IndexDir(dstpath, fs.NewMemRepo(), errorChan)
		assert.Equal(t, srcDir.Info().Strong, dstDir.Info().Strong)
		close(errorChan)
	}()
	for err := range errorChan {
		assert.Tf(t, err == nil, "%v", err)
	}
}

func TestPatchRenameScope(t *testing.T) {
	DoTestPatchRenameScope(t, mkMemRepo)
}

func TestDbPatchRenameScope(t *testing.T) {
	DoTestPatchRenameScope(t, mkDbRepo)
}

func DoTestPatchRenameScope(t *testing.T, mkrepo repoMaker) {
	tg := treegen.New()
	treeSpec := tg.D("foo",
		tg.F("bar", tg.B(6806, 65536)),
		tg.F("baz", tg.B(6806, 65536)))

	srcpath := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(srcpath)
	srcRepo := mkrepo(t)
	defer srcRepo.Close()
	srcStore, err := fs.NewLocalStore(srcpath, srcRepo)
	assert.T(t, err == nil)

	tg = treegen.New()
	treeSpec = tg.D("foo",
		tg.F("baz", tg.B(6806, 65536)),
		tg.F("blop", tg.B(6806, 65536)))

	dstpath := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(dstpath)
	dstRepo := mkrepo(t)
	defer dstRepo.Close()
	dstStore, err := fs.NewLocalStore(dstpath, dstRepo)
	assert.T(t, err == nil)

	patchPlan := NewPatchPlan(srcStore, dstStore)
	//	printPlan(patchPlan)

	failedCmd, err := patchPlan.Exec()
	assert.Tf(t, failedCmd == nil && err == nil, "%v: %v", failedCmd, err)

	errorChan := make(chan os.Error)
	go func() {
		srcDir := fs.IndexDir(srcpath, fs.NewMemRepo(), errorChan)
		dstDir := fs.IndexDir(dstpath, fs.NewMemRepo(), errorChan)
		assert.Equal(t, srcDir.Info().Strong, dstDir.Info().Strong)
		close(errorChan)
	}()
	for err := range errorChan {
		assert.Tf(t, err == nil, "%v", err)
	}
}

func TestPatchPreserveKeeps(t *testing.T) {
	DoTestPatchPreserveKeeps(t, mkMemRepo)
}

func TestDbPatchPreserveKeeps(t *testing.T) {
	DoTestPatchPreserveKeeps(t, mkDbRepo)
}

func DoTestPatchPreserveKeeps(t *testing.T, mkrepo repoMaker) {
	tg := treegen.New()
	treeSpec := tg.D("foo",
		tg.F("bar", tg.B(6806, 65536)),
		tg.F("blop", tg.B(6806, 65536)))

	srcpath := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(srcpath)
	srcRepo := mkrepo(t)
	defer srcRepo.Close()
	srcStore, err := fs.NewLocalStore(srcpath, srcRepo)
	assert.T(t, err == nil)

	tg = treegen.New()
	treeSpec = tg.D("foo",
		tg.F("baz", tg.B(6806, 65536)),
		tg.F("blop", tg.B(6806, 65536)))

	dstpath := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(dstpath)
	dstRepo := mkrepo(t)
	defer dstRepo.Close()
	dstStore, err := fs.NewLocalStore(dstpath, dstRepo)
	assert.T(t, err == nil)

	patchPlan := NewPatchPlan(srcStore, dstStore)
	//	printPlan(patchPlan)

	failedCmd, err := patchPlan.Exec()
	assert.Tf(t, failedCmd == nil && err == nil, "%v: %v", failedCmd, err)

	info, err := os.Stat(filepath.Join(dstpath, "foo", "bar"))
	assert.T(t, err == nil && info != nil)

	info, err = os.Stat(filepath.Join(dstpath, "foo", "blop"))
	assert.T(t, err == nil && info != nil)
}

func TestClean(t *testing.T) {
	DoTestClean(t, mkMemRepo)
}

func TestDbClean(t *testing.T) {
	DoTestClean(t, mkDbRepo)
}

func DoTestClean(t *testing.T, mkrepo repoMaker) {
	tg := treegen.New()
	treeSpec := tg.D("foo",
		tg.D("bar",
			tg.D("aleph",
				tg.F("A", tg.B(42, 65537)),
				tg.F("a", tg.B(42, 65537)))))
	srcpath := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(srcpath)
	srcRepo := mkrepo(t)
	defer srcRepo.Close()
	srcStore, err := fs.NewLocalStore(srcpath, srcRepo)
	assert.T(t, err == nil)

	tg = treegen.New()
	treeSpec = tg.D("foo",
		tg.D("bar",
			tg.D("aleph",
				tg.F("A", tg.B(42, 65537)),
				tg.F("a", tg.B(42, 65537))),
			tg.D("beth",
				tg.F("B", tg.B(43, 65537)),
				tg.F("b", tg.B(43, 65537))),
			tg.D("jimmy",
				tg.F("G", tg.B(44, 65537)),
				tg.F("g", tg.B(44, 65537)))),
		tg.D("baz",
			tg.D("uno",
				tg.F("1", tg.B(1, 65537)),
				tg.F("I", tg.B(1, 65537))),
			tg.D("dos",
				tg.F("2", tg.B(11, 65537)),
				tg.F("II", tg.B(11, 65537))),
			tg.D("tres",
				tg.F("3", tg.B(111, 65537)),
				tg.F("III", tg.B(111, 65537)))))
	dstpath := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(dstpath)
	dstRepo := mkrepo(t)
	defer dstRepo.Close()
	dstStore, err := fs.NewLocalStore(dstpath, dstRepo)
	assert.T(t, err == nil)

	onePath := dstStore.Resolve(filepath.Join("foo", "baz", "uno", "1"))
	_, err = os.Stat(onePath)
	assert.Tf(t, err == nil, "%v", err)

	patchPlan := NewPatchPlan(srcStore, dstStore)
	failedCmd, err := patchPlan.Exec()
	assert.Tf(t, failedCmd == nil, "%v", failedCmd)
	assert.Tf(t, err == nil, "%v", err)

	errors := make(chan os.Error)
	go func() {
		patchPlan.Clean(errors)
		close(errors)
	}()
	for err := range errors {
		assert.Tf(t, err == nil, "%v", err)
	}

	onePath = dstStore.Resolve(filepath.Join("foo", "baz", "uno", "1"))
	_, err = os.Stat(onePath)
	assert.Tf(t, err != nil, "%v", err)
}

func TestSetModeNew(t *testing.T) {
	DoTestSetModeNew(t, mkMemRepo)
}

func TestDbSetModeNew(t *testing.T) {
	DoTestSetModeNew(t, mkDbRepo)
}

func DoTestSetModeNew(t *testing.T, mkrepo repoMaker) {
	tg := treegen.New()
	treeSpec := tg.D("foo",
		tg.D("bar",
			tg.D("aleph",
				tg.F("A", tg.B(42, 65537)),
				tg.F("a", tg.B(42, 65537)))))
	srcpath := treegen.TestTree(t, treeSpec)
	os.Chmod(filepath.Join(srcpath, "foo", "bar", "aleph", "A"), 0765)
	os.Chmod(filepath.Join(srcpath, "foo", "bar"), 0711)
	defer os.RemoveAll(srcpath)
	srcRepo := mkrepo(t)
	defer srcRepo.Close()
	srcStore, err := fs.NewLocalStore(srcpath, srcRepo)
	assert.T(t, err == nil)

	tg = treegen.New()
	treeSpec = tg.D("foo")
	dstpath := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(dstpath)
	dstRepo := mkrepo(t)
	defer dstRepo.Close()
	dstStore, err := fs.NewLocalStore(dstpath, dstRepo)
	assert.T(t, err == nil)

	patchPlan := NewPatchPlan(srcStore, dstStore)
	failedCmd, err := patchPlan.Exec()
	assert.Tf(t, failedCmd == nil, "%v", failedCmd)
	assert.Tf(t, err == nil, "%v", err)

	errors := make(chan os.Error)
	go func() {
		patchPlan.Clean(errors)
		close(errors)
	}()
	for err := range errors {
		assert.Tf(t, err == nil, "%v", err)
	}

	errors = make(chan os.Error)
	go func() {
		patchPlan.SetMode(errors)
		close(errors)
	}()
	for err := range errors {
		assert.Tf(t, err == nil, "%v", err)
	}

	fileinfo, err := os.Stat(filepath.Join(dstpath, "foo", "bar", "aleph", "A"))
	assert.T(t, fileinfo != nil)
	assert.Equal(t, uint32(0765), fileinfo.Permission())

	fileinfo, err = os.Stat(filepath.Join(dstpath, "foo", "bar"))
	assert.T(t, fileinfo != nil)
	assert.Equal(t, uint32(0711), fileinfo.Permission())
}

func TestSetModeOverwrite(t *testing.T) {
	DoTestSetModeOverwrite(t, mkMemRepo)
}

func TestDbSetModeOverwrite(t *testing.T) {
	DoTestSetModeOverwrite(t, mkDbRepo)
}

func DoTestSetModeOverwrite(t *testing.T, mkrepo repoMaker) {
	tg := treegen.New()
	treeSpec := tg.D("foo",
		tg.D("bar",
			tg.D("aleph",
				tg.F("A", tg.B(42, 65537)),
				tg.F("a", tg.B(42, 65537)))))
	srcpath := treegen.TestTree(t, treeSpec)
	os.Chmod(filepath.Join(srcpath, "foo", "bar", "aleph", "A"), 0765)
	os.Chmod(filepath.Join(srcpath, "foo", "bar"), 0711)
	defer os.RemoveAll(srcpath)
	srcRepo := mkrepo(t)
	defer srcRepo.Close()
	srcStore, err := fs.NewLocalStore(srcpath, srcRepo)
	assert.T(t, err == nil)

	tg = treegen.New()
	treeSpec = tg.D("foo",
		tg.D("bar",
			tg.D("aleph",
				tg.F("A", tg.B(42, 65537)),
				tg.F("a", tg.B(42, 65537)))))
	dstpath := treegen.TestTree(t, treeSpec)
	os.Chmod(filepath.Join(dstpath, "foo", "bar", "aleph", "A"), 0600)
	os.Chmod(filepath.Join(dstpath, "foo", "bar"), 0700)
	defer os.RemoveAll(dstpath)
	dstRepo := mkrepo(t)
	defer dstRepo.Close()
	dstStore, err := fs.NewLocalStore(dstpath, dstRepo)
	assert.T(t, err == nil)

	patchPlan := NewPatchPlan(srcStore, dstStore)
	failedCmd, err := patchPlan.Exec()
	assert.Tf(t, failedCmd == nil, "%v %v", failedCmd, err)
	assert.Tf(t, err == nil, "%v", err)

	errors := make(chan os.Error)
	go func() {
		patchPlan.Clean(errors)
		close(errors)
	}()
	for err := range errors {
		assert.Tf(t, err == nil, "%v", err)
	}

	errors = make(chan os.Error)
	go func() {
		patchPlan.SetMode(errors)
		close(errors)
	}()
	for err := range errors {
		assert.Tf(t, err == nil, "%v", err)
	}

	fileinfo, err := os.Stat(filepath.Join(dstpath, "foo", "bar", "aleph", "A"))
	assert.T(t, fileinfo != nil)
	assert.Equal(t, uint32(0765), fileinfo.Permission())

	fileinfo, err = os.Stat(filepath.Join(dstpath, "foo", "bar"))
	assert.T(t, fileinfo != nil)
	assert.Equal(t, uint32(0711), fileinfo.Permission())
}
