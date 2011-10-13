
/*

Let's invent a mini internal DSL for generating test data. It will look like:

D("",
	D("",
		F("",
			B(1232, 50),
			B(121212, 1000)),
		F("",
			B(1, 1),
			B(2, 2))))

Directory structure is pretty much what you'd expect.

Directory and file names are randomly generated strings when empty.

File contents specified with one or more expr of the form (LENGTH SEED),
where LENGTH is number of random bytes to generate, and SEED is the PRNG 
seed used to start the generator.

This allows us to specify a repeatable sequence of bytes at an 
arbitrary location in the generated file in a very compact way,
useful for testing match & patch.

*/

package treegen

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"rand"
	"testing"
	
	"github.com/bmizerany/assert"
)

type Generated interface {}

type Dir struct {
	Name string
	Contents []Generated
}

type File struct {
	Name string
	Contents []Generated
}

type Bytes struct {
	Seed int64
	Length int64
}

type TreeGen struct {
	rand *rand.Rand
}

func New() *TreeGen {
	return &TreeGen{ rand: rand.New(rand.NewSource(42)) }
}

func (treeGen *TreeGen) randomName() string {
	len := treeGen.rand.Intn(17) + 3
	buf := &bytes.Buffer{}
	
	for i := 0; i < len; i++ {
		buf.WriteByte(byte(0x20 + treeGen.rand.Intn(0x7e-0x20)))
	}
	
	return string(buf.Bytes())
}

func (treeGen *TreeGen) D(name string, contents ...Generated) *Dir {
	if name == "" {
		name = treeGen.randomName()
	}
	return &Dir{Name: name, Contents: contents}
}

func (treeGen *TreeGen) F(name string, contents ...Generated) *File {
	if name == "" {
		name = treeGen.randomName()
	}
	return &File{Name: name, Contents: contents}
}

func (treeGen *TreeGen) B(seed int64, length int64) *Bytes {
	return &Bytes{Seed: seed, Length: length}
}

const PREFIX string = "treegen"

func TestTree(t *testing.T, g Generated) string {
	tempdir, err := ioutil.TempDir("", PREFIX)
	assert.Tf(t, err == nil, "Fail to create temp dir")
	
	err = Fab(tempdir, g)
	assert.Tf(t, err == nil, "Fail to fab tree")
	
	return tempdir
}

func Fab(parent string, g Generated) (os.Error) {
	if d, isD := g.(*Dir); isD {
		return d.fab(parent)
	} else if f, isF := g.(*File); isF {
		return f.fab(parent)
	} else if b, isB := g.(*Bytes); isB {
		return b.fab(parent)
	}
	
	return os.NewError(fmt.Sprintf("WTF is this: %v?", g))
}

func (d *Dir) fab(parent string) (os.Error) {
	name := d.Name
	
	path := filepath.Join(parent, name)
	err := os.Mkdir(path, 0777)
	if err != nil { return err }
	
	return fabEntries(path, d.Contents[0], d.Contents[1:])
}

func (f *File) fab(parent string) (os.Error) {
	name := f.Name
	
	path := filepath.Join(parent, name)
	fh, err := os.Create(path)
	if fh == nil { return err }
	fh.Close()
	
	return fabEntries(path, f.Contents[0], f.Contents[1:])
}

const CHUNKSIZE int = 8192

func (b *Bytes) fab(parent string) (os.Error) {
	fh, err := os.OpenFile(parent, os.O_RDWR | os.O_APPEND, 0644)
	if fh == nil { return err }
	defer fh.Close()
	
	rnd := rand.New(rand.NewSource(b.Seed))
	
	for toWrite := b.Length; toWrite > 0; {
		buf := &bytes.Buffer{}
		
		for i := 0; i < CHUNKSIZE && toWrite > 0; i++ {
			buf.WriteByte(byte(rnd.Int()))
			toWrite--;
		}
		
		_, err = buf.WriteTo(fh)
		if err != nil { return err }
	}
	
	return nil
}

func fabEntries(path string, first Generated, rest []Generated) (os.Error) {
	if err := Fab(path, first); err != nil {
		return err
	}
	
	if (len(rest) == 0) {
		return nil
	}
	
	return fabEntries(path, rest[0], rest[1:])
}


