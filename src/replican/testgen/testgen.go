
/*

Let's invent a mini internal DSL for generating test data. It will look like:

&D{Contents: {
	&D{Contents: {
		&F{Contents: {
			&B{Seed: 1232, N: 50},
			&B{Seed: 121212, N: 1000}}},
		&F{Contents: {
			&B{Seed: 1, N: 1},
			&B{Seed: 2, N: 2}}}}}}

Directory structure is pretty much what you'd expect.

Directory and file names are randomly generated strings unless otherwise specified.

File contents specified with one or more expr of the form (LENGTH SEED),
where LENGTH is number of random bytes to generate, and SEED is the PRNG 
seed used to start the generator.

This allows us to specify a repeatable sequence of bytes at an 
arbitrary location in the generated file in a very compact way,
useful for testing match & patch.

*/

package testgen

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"rand"
)

type N interface {}

type D struct {
	Contents []N
	Name string
}

type F struct {
	Contents []N
	Name string
}

type B struct {
	Seed int64
	N int64
}

func Fab(parent string, n N) (os.Error) {
	if d, isD := n.(*D); isD {
		return d.fab(parent)
	} else if f, isF := n.(*F); isF {
		return f.fab(parent)
	} else if b, isB := n.(*B); isB {
		return b.fab(parent)
	}
	
	return os.NewError(fmt.Sprintf("WTF is this: %v?", n))
}

func (d *D) fab(parent string) (os.Error) {
	name := d.Name
	if name == "" { name = RandomName() }
	
	path := filepath.Join(parent, name)
	err := os.Mkdir(path, 0777)
	if err != nil { return err }
	
	return fabEntries(path, d.Contents[0], d.Contents[1:])
}

func (f *F) fab(parent string) (os.Error) {
	name := f.Name
	if name == "" { name = RandomName() }
	
	path := filepath.Join(parent, name)
	fh, err := os.Create(path)
	if fh == nil { return err }
	fh.Close()
	
	return fabEntries(path, f.Contents[0], f.Contents[1:])
}

const CHUNKSIZE int = 8192

func (b *B) fab(parent string) (os.Error) {
	fh, err := os.Open(parent)
	if fh == nil { return err }
	defer fh.Close()
	
	rnd := rand.New(rand.NewSource(b.Seed))
	
	for toWrite := b.N; toWrite > 0; {
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

func fabEntries(path string, first N, rest []N) (os.Error) {
	if err := Fab(path, first); err != nil {
		return err
	}
	
	if (len(rest) == 0) {
		return nil
	}
	
	return fabEntries(path, rest[0], rest[1:])
}

func RandomName() string {
	buf := &bytes.Buffer{}
	rnd := rand.New(rand.NewSource(42))
	len := rnd.Intn(17) + 3
	
	for i := 0; i < len; i++ {
		buf.WriteByte(byte(0x20 + rnd.Intn(0x7e-0x20)))
	}
	
	return string(buf.Bytes())
}



