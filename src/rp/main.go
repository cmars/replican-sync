
package main

import (
	"replican/fs"
	"replican/merge"
	"os"
	"fmt"
	
	"optarg.googlecode.com/hg/optarg"
)

func main() {
	verboseOpt := optarg.NewBoolOption("v", "verbose")
	
	files, err := optarg.Parse()
	if err != nil { 
		fmt.Fprintf(os.Stderr, "%s\n", err)
		optarg.Usage()
		os.Exit(1)
	}
	
	if (len(files) < 2) {
		fmt.Printf("Usage: %s <src> <dst>\n", os.Args[0])
		os.Exit(1)
	}
	
	srcpath := files[0]
	dstpath := files[1]
	
	srcStore, err := fs.NewLocalStore(srcpath)
	if err != nil { die(fmt.Sprintf("Failed to read source %s", srcpath), err) }
	
	dstStore, err := fs.NewLocalStore(dstpath)
	if err != nil { die(fmt.Sprintf("Failed to read destination %s", srcpath), err) }
	
	patchPlan := merge.NewPatchPlan(srcStore, dstStore)
	
	if verboseOpt.Value {
		fmt.Printf("%v\n", patchPlan)
	}
	
	failedCmd, err := patchPlan.Exec()
	if err != nil { die(failedCmd.String(), err) }
	
	os.Exit(0)
}

func die(message string, err os.Error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", message, err)
	os.Exit(1)
}



