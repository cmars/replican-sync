package main

import (
	"bytes"
	"fmt"
	"rand"
	"github.com/cmars/replican-sync/replican/fs"
)

/*
 * Create a pseudo-random block using given seed.
 */
func randBuf(seed int64) []byte {
	rnd := rand.New(rand.NewSource(seed))

	buf := &bytes.Buffer{}

	for i := 0; i < fs.BLOCKSIZE; i++ {
		buf.WriteByte(byte(rnd.Int()))
	}

	return buf.Bytes()
}

/*
 * Trainwreck finds blocks with weak checksum collisions.
 */
func main() {
	weaks := make(map[int]int64)
	nhits := 0
	for i := int64(0); i < int64(0xFFFFFFFF); i++ {
		buf := randBuf(i)
		block := fs.IndexBlock(buf)

		collision, matched := weaks[block.Weak()]
		if matched {
			fmt.Printf("Collision found: seeds %d and %d\n", collision, i)
			nhits++
		} else {
			weaks[block.Weak()] = i
		}
	}

	if nhits == 0 {
		fmt.Printf("No collisions found, try something else.\n")
	}
}
