
package merge

import (
	"os"
	"replican/blocks"
	"testing"
)

func TestMatchIdentity(t *testing.T) {
	srcPath := "./testroot/My Music/0 10k 30.mp4"
	dstPath := srcPath
	
	match, err := Match(srcPath, dstPath)
	
	if err != nil {
		t.Fatalf("Error matching files: %s", err.String())
	}
	
	nMatches := 0
	for i, match := range match.BlockMatches {
		if match.DstOffset % int64(blocks.BLOCKSIZE) != 0 {
			t.Errorf("Destination match block# %d not aligned with blocksize! (offset=%d)",
				i, match.DstOffset)
		}
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
		
		if nExpectMatches != int64(nMatches) {
			t.Errorf("Expected %d matches, got %d", nExpectMatches, nMatches)
		}
	}
	
	lastBlockSize := fileInfo.Size - int64(match.BlockMatches[14].DstOffset)
	if lastBlockSize != 5419 {
		t.Errorf("Unxpected last block size: %d", lastBlockSize)
	}
}

func TestMatchMunge(t *testing.T) {
	srcPath := "./testroot/My Music/0 10k 30.mp4"
	dstPath := "./testroot/My Music/0 10k 30 munged.mp4"
	
	match, err := Match(srcPath, dstPath)
	
	if err != nil {
		t.Fatalf("Error matching files: %s", err.String())
	}
	
	nMatches := 0
	for i, match := range match.BlockMatches {
		if match.DstOffset % int64(blocks.BLOCKSIZE) != 0 {
			t.Errorf("Destination match block# %d not aligned with blocksize! (offset=%d)",
				i, match.DstOffset)
		}
		nMatches++
	}
	
	const nExpectedMatches = 13
	if nMatches != nExpectedMatches {
		t.Errorf("Expected %d matches, got %d", nExpectedMatches, nMatches)
	}
}




