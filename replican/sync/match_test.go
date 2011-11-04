package sync

import (
	"github.com/cmars/replican-sync/replican/fs"
	"os"
	"testing"

	"github.com/bmizerany/assert"
)

// Test that the matcher matches all blocks in two identical files.
func TestMatchIdentity(t *testing.T) {
	srcPath := "../../testroot/My Music/0 10k 30.mp4"
	dstPath := srcPath

	match, err := Match(srcPath, dstPath)
	assert.Tf(t, err == nil, "%v", err)

	nMatches := 0
	for i, match := range match.BlockMatches {
		assert.Equalf(t, int64(0), match.DstOffset%int64(fs.BLOCKSIZE),
			"Destination match block# %d not aligned with blocksize! (offset=%d)",
			i, match.DstOffset)
		nMatches++
	}

	fileInfo, err := os.Stat(srcPath)
	if fileInfo == nil {
		t.Fatalf("Cannot stat file %s: ", err.String())
	} else {
		nExpectMatches := fileInfo.Size / int64(fs.BLOCKSIZE)
		if fileInfo.Size%int64(fs.BLOCKSIZE) > 0 {
			nExpectMatches++
		}

		assert.Equal(t, nExpectMatches, int64(nMatches))
	}

	lastBlockSize := fileInfo.Size - int64(match.BlockMatches[14].DstOffset)
	assert.Equalf(t, int64(5419), lastBlockSize,
		"Unxpected last block size: %d", lastBlockSize)
}

// Test that the matcher matches blocks properly between two different files.
// The munged file has a few bytes changed at known offsets which we check for.
func TestMatchMunge(t *testing.T) {
	srcPath := "../../testroot/My Music/0 10k 30.mp4"
	dstPath := "../../testroot/My Music/0 10k 30 munged.mp4"

	match, err := Match(srcPath, dstPath)

	assert.T(t, err == nil)

	nMatches := 0
	for i, match := range match.BlockMatches {
		assert.Equalf(t, int64(0), match.DstOffset%int64(fs.BLOCKSIZE),
			"Destination match block# %d not aligned with blocksize! (offset=%d)",
			i, match.DstOffset)
		nMatches++
	}

	assert.Equal(t, 13, nMatches)

	notMatches := match.NotMatched()
	assert.Equal(t, 2, len(notMatches))
}

// Test some corner cases of FileMatch.NotMatched to correctly 
// identify unmatched ranges between two files. No files were harmed 
// in the creation of this test, we're fabricating a fake FileMatch.
func TestHoles(t *testing.T) {
	testMatch := &FileMatch{
		SrcSize: 99099, DstSize: 99099,
		BlockMatches: []*BlockMatch{
			&BlockMatch{DstOffset: 123},
			&BlockMatch{DstOffset: 9991},
			&BlockMatch{DstOffset: 23023},
		}}

	notMatched := testMatch.NotMatched()

	assert.Tf(t, len(notMatched) > 0, "Failed to detect obvious holes in match")

	for i, unmatch := range notMatched {
		switch i {
		case 0:
			assert.Equal(t, int64(0), unmatch.From)
			assert.Equal(t, int64(123), unmatch.To)
			break

		case 1:
			assert.Equal(t, int64(8315), unmatch.From)
			assert.Equal(t, int64(9991), unmatch.To)
			break

		case 2:
			assert.Equal(t, int64(18183), unmatch.From)
			assert.Equal(t, int64(23023), unmatch.To)
			break

		case 3:
			assert.Equal(t, int64(31215), unmatch.From)
			assert.Equal(t, int64(99099), unmatch.To)
			break

		default:
			t.Fatalf("Unexpected not-match %v", unmatch)
		}
	}
}
