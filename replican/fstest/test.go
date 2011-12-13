package fstest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmizerany/assert"
	"github.com/cmars/replican-sync/replican/treegen"
	"github.com/cmars/replican-sync/replican/fs"
)

func DoTestDirIndex(t *testing.T, repo fs.NodeRepo) {
	dir := fs.IndexDir("testroot/", repo, nil)
	assert.T(t, dir != nil)
	
	assert.Equal(t, 2, len(dir.SubDirs()))
	assert.Equal(t, 4, len(dir.Files()))
	
	assert.Equal(t, "feab33f9685531a1c1c9c22d5d8af98267ca9426", dir.Info().Strong)

	var myMusic fs.Dir = dir.SubDirs()[0]
	assert.Equal(t, "My Music", myMusic.Name())

	for i := 0; i < 2; i++ {
		var mp4file fs.FsNode = myMusic.Files()[i]
		assert.Tf(t, strings.HasPrefix(mp4file.Name(), "0 10k 30"),
			"Unexpected d -> d -> f name: %s", mp4file.Name())
	}
}

func DoTestVisitDirsOnly(t *testing.T, repo fs.NodeRepo) {
	dir := fs.IndexDir("../../testroot/", repo, nil)
	assert.T(t, dir != nil)

	collect := []fs.Dir{}
	visited := []fs.Node{}

	fs.Walk(dir, func(node fs.Node) bool {
		visited = append(visited, node)

		d, ok := node.(fs.Dir)
		if ok {
			collect = append(collect, d)
			return true
		}

		_, ok = node.(fs.File)
		if ok {
			return false
		}

		t.Errorf("Unexpected type during visit: %v", node)
		return true
	})

	assert.Equalf(t, 3, len(collect), "Unexpected dirs in testroot/: %v", collect)

	for _, node := range visited {
		_, ok := node.(fs.Block)
		if ok {
			t.Fatalf("Should not have gotten a block, we told visitor to stop at file level.")
		}
	}
}

func DoTestVisitBlocks(t *testing.T, repo fs.NodeRepo) {
	dir := fs.IndexDir("../../testroot/", repo, nil)
	assert.T(t, dir != nil)

	collect := []fs.Block{}

	fs.Walk(dir, func(node fs.Node) bool {
		b, ok := node.(fs.Block)
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

func DoTestNodeRelPath(t *testing.T, repo fs.NodeRepo) {
	tg := treegen.New()
	treeSpec := tg.D("foo", tg.F("bar", tg.B(42, 65537)))

	path := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(path)

	dir := fs.IndexDir(path, repo, nil)

	assert.Equal(t, "", fs.RelPath(dir))
	assert.Equal(t, "foo", fs.RelPath(dir.SubDirs()[0]))
	assert.Equal(t, filepath.Join("foo", "bar"), fs.RelPath(dir.SubDirs()[0].Files()[0]))

	assert.Equal(t, filepath.Join("foo", "bar"), fs.RelPath(dir.SubDirs()[0].Files()[0]))
}

func DoTestStoreRelPath(t *testing.T, repo fs.NodeRepo) {
	tg := treegen.New()
	treeSpec := tg.D("foo", tg.F("bar", tg.B(42, 65537)))

	path := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(path)

	store, err := fs.NewLocalStore(path, repo)
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

func DoTestDirResolve(t *testing.T, repo fs.NodeRepo) {
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

	foo := fs.IndexDir(filepath.Join(path, "foo"), repo, nil)

	var node fs.FsNode
	var found bool

	node, found = fs.Lookup(foo, "bar")
	assert.T(t, found)
	_, isDir := node.(fs.Dir)
	assert.T(t, isDir)

	node, found = fs.Lookup(foo, filepath.Join("bar", "aleph"))
	assert.T(t, found)
	_, isDir = node.(fs.Dir)
	assert.T(t, isDir)

	node, found = fs.Lookup(foo, filepath.Join("bar", "aleph", "A"))
	assert.T(t, found)
	_, isFile := node.(fs.File)
	assert.T(t, isFile)
}

func DoTestDirDescent(t *testing.T, repo fs.NodeRepo) {
	tg := treegen.New()
	treeSpec := tg.D("foo",
		tg.F("baobab", tg.B(91, 65537)),
		tg.D("bar",
			tg.D("aleph",
				tg.F("a", tg.B(42, 65537)))),
		tg.F("bar3003", tg.B(777, 65537)))

	path := treegen.TestTree(t, treeSpec)
	defer os.RemoveAll(path)

	dir := fs.IndexDir(path, repo, nil)

	for _, fpath := range []string{
		filepath.Join("foo", "baobab"),
		filepath.Join("foo", "bar", "aleph", "a"),
		filepath.Join("foo", "bar3003")} {
		node, found := fs.Lookup(dir, fpath)
		assert.Tf(t, found, "not found: %s", fpath)
		_, isFile := node.(fs.File)
		assert.T(t, isFile)
	}

	node, found := fs.Lookup(dir, filepath.Join("foo", "bar"))
	assert.T(t, found)
	_, isDir := node.(fs.Dir)
	assert.T(t, isDir)
}

func DoTestParentRefs(t *testing.T, repo fs.NodeRepo) {
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

	foo := fs.IndexDir(filepath.Join(path, "foo"), repo, nil)
	rootCount := 0
	fs.Walk(foo, func(node fs.Node) bool {
		switch node.(type) {
		case fs.Dir:
			dir := node.(fs.Dir)
			if _, hasParent := dir.Parent(); !hasParent {
				rootCount++
			}
			break
		case fs.File:
			file := node.(fs.File)
			_, hasParent := file.Parent()
			assert.Tf(t, hasParent, "%v is root?!", file.Info())
			break
		case fs.Block:
			block := node.(fs.Block)
			_, hasParent := block.Parent()
			assert.Tf(t, hasParent, "%v is root?!", block.Info())
			break
		}
		return true
	})

	assert.Equal(t, 1, rootCount)
}
