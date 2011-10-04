
package blocks

import (
	"fmt"
	"os"
	"crypto/sha1"
)

const BLOCKSIZE uint = 8192

type WeakChecksum struct {
	a uint
	b uint
}

func (weak *WeakChecksum) Write(buf []byte) {
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
	var sha1 = sha1.New()
	
	for {
		switch rd, err := f.Read(buf[:]); true {
		case rd < 0:
			return nil, err
		case rd == 0:
			file.Strong = fmt.Sprintf("%x", sha1.Sum())
			return file, nil
		case rd > 0:
			// update block hashes
			block = IndexBlock(buf[0:rd])
			file.Blocks = append(file.Blocks, block)
			
			// update file hash
			sha1.Write(buf[0:rd])
		}
	}
	
	return nil, nil
}

func IndexBlock(buf []byte) (block *Block) {
	block = new(Block)
	
	var weak = new(WeakChecksum)
	weak.Write(buf)
	block.Weak = weak.Get()
	
	var sha1 = sha1.New()
	sha1.Write(buf)
	block.Strong = fmt.Sprintf("%x", sha1.Sum())
	
	return block
}

type Node interface {
	IsRoot() bool
	Strong() string
	Parent() Node
	Children() []Node
}

type Block struct {
	Weak uint
	Strong string
}

type File struct {
	Name string
	Strong string
	Blocks []*Block
}

type Dir struct {
	Name string
}



