
package merge

import (
	"os"
	"replican/blocks"
	"testing"
	"github.com/bmizerany/assert"
)

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
		
		assert.Equalf(t, nExpectMatches, int64(nMatches),
				"Expected %d matches, got %d", nExpectMatches, nMatches)
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
	
	const nExpectedMatches = 13
	assert.Equalf(t, nExpectedMatches, nMatches, 
			"Expected %d matches, got %d", nExpectedMatches, nMatches)
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
			assert.Equal(t, int64(122), unmatch.To)
			break
			
		case 1:
			assert.Equal(t, int64(8315), unmatch.From)
			assert.Equal(t, int64(9990), unmatch.To)
			break
			
		case 2:
			assert.Equal(t, int64(18183), unmatch.From)
			assert.Equal(t, int64(23022), unmatch.To)
			break
			
		case 3:
			assert.Equal(t, int64(31215), unmatch.From)
			assert.Equal(t, int64(99098), unmatch.To)
			break
		
		default:
			t.Fatalf("Unexpected not-match %v", unmatch)
		}
	}
}



