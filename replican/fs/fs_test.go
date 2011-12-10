package fs

import (
	"os"
	"path/filepath"
	"github.com/cmars/replican-sync/replican/treegen"
	"strings"
	"testing"

	"github.com/bmizerany/assert"
)

func TestIndexSomeMp3(t *testing.T) {
	cwd, _ := os.Getwd()
	t.Logf("CWD=%s", cwd)

	f, blks, err := IndexFile("../../testroot/My Music/0 10k 30.mp4")
	if f == nil {
		t.Fatalf("Failed to index file: %s", err.String())
	}

	assert.Equal(t, "5ab3e5d621402e5894429b5f595a1e2d7e1b3078", f.Strong)
	assert.Equal(t, "d1f11a93449fa4d3f320234743204ce157bbf1f3", blks[0].Strong)
	assert.Equal(t, "eabbe570b21cd2c5101a18b51a3174807fa5c0da", blks[1].Strong)
}

func TestDirIndex(t *testing.T) {
	DoTestDirIndex(t, NewMemRepo())
}

func DoTestDirIndex(t *testing.T, repo NodeRepo) {
	dir := IndexDir("testroot/", repo, nil)

	assert.Equal(t, "feab33f9685531a1c1c9c22d5d8af98267ca9426", dir.Info().Strong)

	var myMusic Dir = dir.SubDirs()[0]
	assert.Equal(t, "My Music", myMusic.Name())

	for i := 0; i < 2; i++ {
		var mp4file FsNode = myMusic.Files()[i]
		assert.Tf(t, strings.HasPrefix(mp4file.Name(), "0 10k 30"),
			"Unexpected d -> d -> f name: %s", mp4file.Name())
	}
}

func TestVisitDirsOnly(t *testing.T) {
	DoTestVisitDirsOnly(t, NewMemRepo())
}

func DoTestVisitDirsOnly(t *testing.T, repo NodeRepo) {
	dir := IndexDir("../../testroot/", repo, nil)

	collect := []Dir{}
	visited := []Node{}

	Walk(dir, func(node Node) bool {
		visited = append(visited, node)

		d, ok := node.(Dir)
		if ok {
			collect = append(collect, d)
			return true
		}

		_, ok = node.(File)
		if ok {
			return false
		}

		t.Errorf("Unexpected type during visit: %v", node)
		return true
	})

	assert.Equalf(t, 3, len(collect), "Unexpected dirs in testroot/: %v", collect)

	for _, node := range visited {
		_, ok := node.(Block)
		if ok {
			t.Fatalf("Should not have gotten a block, we told visitor to stop at file level.")
		}
	}
}

func TestVisitBlocks(t *testing.T) {
	DoTestVisitBlocks(t, NewMemRepo())
}

func DoTestVisitBlocks(t *testing.T, repo NodeRepo) {
	dir := IndexDir("../../testroot/", repo, nil)

	collect := []Block{}

	Walk(dir, func(node Node) bool {
		b, ok := node.(Block)
		if ok {
			collect = append(collect, b)
		}

		return true
	})

	matched := false
	for _, block := range collect {
		if block.Info().Strong == "d1f11a93449fa4d3f320234743204ce157bbf1f3" {
			matched = true
		}
	}

	assert.Tf(t, matched, "Failed to find expected block")
}

func TestNodeRelPath(t *testing.T) {
	DoTestNodeRelPath(t, NewMemRepo())
}

func DoTestNodeRelPath(t *testing.T, repo NodeRepo) {
	tg := treegen.New()
	treeSpec := tg.D("foo", tg.F("bar", tg.B(42, 65537)))

	path := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(path)

	dir := IndexDir(path, repo, nil)

	assert.Equal(t, "", RelPath(dir))
	assert.Equal(t, "foo", RelPath(dir.SubDirs()[0]))
	assert.Equal(t, filepath.Join("foo", "bar"), RelPath(dir.SubDirs()[0].Files()[0]))

	assert.Equal(t, filepath.Join("foo", "bar"), RelPath(dir.SubDirs()[0].Files()[0]))
}

func TestStoreRelPath(t *testing.T) {
	DoTestStoreRelPath(t, NewMemRepo())
}

func DoTestStoreRelPath(t *testing.T, repo NodeRepo) {
	tg := treegen.New()
	treeSpec := tg.D("foo", tg.F("bar", tg.B(42, 65537)))

	path := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(path)

	store, err := NewLocalStore(path, repo)
	assert.T(t, err == nil)

	relFoo := store.RelPath(filepath.Join(path, "foo"))
	assert.Equalf(t, "foo", relFoo, "'%v': not a foo", relFoo)

	// Relocate bar
	newBar, err := store.Relocate(filepath.Join(filepath.Join(path, "foo"), "bar"))
	assert.T(t, err == nil)

	// new bar's parent should still be foo
	newBarParent, _ := filepath.Split(newBar)
	newBarParent = strings.TrimRight(newBarParent, "/\\")

	// old bar should not exist
	_, err = os.Stat(filepath.Join(filepath.Join(path, "foo"), "bar"))
	assert.T(t, err != nil)

	foobar := filepath.Join("foo", "bar")
	assert.Equal(t, newBar, store.Resolve(foobar), "reloc path %s != resolve foo/bar %s",
		newBar, store.Resolve(foobar))
}

func TestDirResolve(t *testing.T) {
	DoTestDirResolve(t, NewMemRepo())
}

func DoTestDirResolve(t *testing.T, repo NodeRepo) {
	tg := treegen.New()
	treeSpec := tg.D("foo",
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

	path := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(path)

	foo := IndexDir(filepath.Join(path, "foo"), repo, nil)

	var node FsNode
	var found bool

	node, found = Lookup(foo, "bar")
	assert.T(t, found)
	_, isDir := node.(Dir)
	assert.T(t, isDir)

	node, found = Lookup(foo, filepath.Join("bar", "aleph"))
	assert.T(t, found)
	_, isDir = node.(Dir)
	assert.T(t, isDir)

	node, found = Lookup(foo, filepath.Join("bar", "aleph", "A"))
	assert.T(t, found)
	_, isFile := node.(File)
	assert.T(t, isFile)
}

func TestDirDescent(t *testing.T) {
	DoTestDirDescent(t, NewMemRepo())
}

func DoTestDirDescent(t *testing.T, repo NodeRepo) {
	tg := treegen.New()
	treeSpec := tg.D("foo",
		tg.F("baobab", tg.B(91, 65537)),
		tg.D("bar",
			tg.D("aleph",
				tg.F("a", tg.B(42, 65537)))),
		tg.F("bar3003", tg.B(777, 65537)))

	path := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(path)

	dir := IndexDir(path, repo, nil)

	for _, fpath := range []string{
		filepath.Join("foo", "baobab"),
		filepath.Join("foo", "bar", "aleph", "a"),
		filepath.Join("foo", "bar3003")} {
		node, found := Lookup(dir, fpath)
		assert.Tf(t, found, "not found: %s", fpath)
		_, isFile := node.(File)
		assert.T(t, isFile)
	}

	node, found := Lookup(dir, filepath.Join("foo", "bar"))
	assert.T(t, found)
	_, isDir := node.(Dir)
	assert.T(t, isDir)
}

func TestParentRefs(t *testing.T) {
	DoTestParentRefs(t, NewMemRepo())
}

func DoTestParentRefs(t *testing.T, repo NodeRepo) {
	tg := treegen.New()
	treeSpec := tg.D("foo",
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

	path := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(path)

	foo := IndexDir(filepath.Join(path, "foo"), repo, nil)
	rootCount := 0
	Walk(foo, func(node Node) bool {
		switch node.(type) {
		case Dir:
			dir := node.(Dir)
			if _, hasParent := dir.Parent(); !hasParent {
				rootCount++
			}
			break
		case File:
			file := node.(File)
			_, hasParent := file.Parent()
			assert.Tf(t, hasParent, "%v is root?!", file.Info())
			break
		case Block:
			block := node.(Block)
			_, hasParent := block.Parent()
			assert.Tf(t, hasParent, "%v is root?!", block.Info())
			break
		}
		return true
	})

	assert.Equal(t, 1, rootCount)
}
