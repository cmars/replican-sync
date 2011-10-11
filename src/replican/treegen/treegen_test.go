
package treegen

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	
	"github.com/bmizerany/assert"
)

const PREFIX string = "treegen"

func TestSimple1(t *testing.T) {
	tempdir, err := ioutil.TempDir("", PREFIX)
	assert.Tf(t, err == nil, "Fail to create temp dir")
	defer os.RemoveAll(tempdir)
	
	tg := New()
	treeSpec := tg.D("", tg.F("", tg.B(42, 65537)))
	
	err = Fab(tempdir, treeSpec)
	if err != nil { fmt.Printf(err.String()) }
	assert.Tf(t, err == nil, "Failed to create tree")
}



