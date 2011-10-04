
package blocks

import (
	"os"
)

const BLOCKSIZE uint = 8192

type WeakChecksum struct {
	a uint
	b uint
}

func (weak *WeakChecksum) Update(buf []byte) {
	for i := 0; i < len(buf); i++ {
		weak.a += uint(buf[i]);
		weak.b += uint(len(buf) - i) * uint(buf[i]);
	}
}

func (weak *WeakChecksum) Get() (uint) {
	return weak.b << 16 | weak.a;
}

func (weak *WeakChecksum) Roll(removedByte byte, newByte byte) {
    weak.a -= uint(removedByte - newByte);
    weak.b -= uint(removedByte) * BLOCKSIZE - weak.a;
}
    
func IndexFile(path string) (file *File, err os.Error) {
	var f *os.File
	var buf [BLOCKSIZE]byte
	
	f, err = os.Open(path)
	if f == nil {
		return nil, err
	}
	
	file = new(File)
	var block *Block
	
	for {
		switch rd, err := f.Read(buf[:]); true {
		case rd < 0:
			return nil, err
		case rd == 0:
			return file, nil
		case rd > 0:
			// update block hashes
			block = IndexBlock(buf[0:rd])
			file.blocks = append(file.blocks, block)
		}
	}
	
	return nil, nil
}

func IndexBlock(buf []byte) (block *Block) {
	block = new(Block)
	var weak = new(WeakChecksum)
	weak.Update(buf)
	block.weak = weak.Get()
	return block
}

type Node interface {
	IsRoot() bool
	Strong() string
	Parent() Node
	Children() []Node
}

type Block struct {
	weak uint
	strong string
}

type File struct {
	name string
	blocks []*Block
}

type Dir struct {
	name string
}



