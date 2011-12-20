package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/cmars/replican-sync/replican/fs"
	"github.com/cmars/replican-sync/replican/fs/sqlite3"
	"github.com/cmars/replican-sync/replican/sync"

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

	if len(files) < 2 {
		die(fmt.Sprintf("Usage: %s <src> <dst>", os.Args[0]), nil)
	}

	srcpath := files[0]
	dstpath := files[1]

	srcinfo, err := os.Stat(srcpath)
	if err != nil {
		die(fmt.Sprintf("Cannot read <src> %s:", srcpath), err)
	}

	dstinfo, err := os.Stat(dstpath)
	if err == os.EEXIST && srcinfo.IsDirectory() {
		os.MkdirAll(dstpath, 0755)
	} else if err == nil && srcinfo.IsDirectory() != dstinfo.IsDirectory() {
		die(fmt.Sprintf(
			"Cannot sync %s to %s: one of these things is not like the other",
			srcinfo, dstinfo), nil)
	}

	srcDbF, err := ioutil.TempFile("", "srcdb")
	if err != nil {
		die("Failed to create source index database", err)
	}
	srcDbF.Close()
	defer os.RemoveAll(srcDbF.Name())
	srcRepo, err := sqlite3.NewDbRepo(srcDbF.Name())
	if err != nil {
		die("Failed to create source index database", err)
	}

	srcStore, err := fs.NewLocalStore(srcpath, srcRepo)
	if err != nil {
		die(fmt.Sprintf("Failed to read source %s", srcpath), err)
	}

	dstDbF, err := ioutil.TempFile("", "dstdb")
	if err != nil {
		die("Failed to create destination index database", err)
	}
	defer os.RemoveAll(dstDbF.Name())
	dstRepo, err := sqlite3.NewDbRepo(dstDbF.Name())
	if err != nil {
		die("Failed to create source index database", err)
	}

	dstStore, err := fs.NewLocalStore(dstpath, dstRepo)
	if err != nil {
		die(fmt.Sprintf("Failed to read destination %s", srcpath), err)
	}

	patchPlan := sync.NewPatchPlan(srcStore, dstStore)

	if verboseOpt.Value {
		fmt.Printf("%v\n", patchPlan)
	}

	failedCmd, err := patchPlan.Exec()
	if err != nil {
		die(failedCmd.String(), err)
	}

	os.Exit(0)
}

func die(message string, err os.Error) {
	if err == nil {
		fmt.Fprint(os.Stderr, message)
	} else {
		fmt.Fprintf(os.Stderr, "%s: %v\n", message, err)
	}
	os.Exit(1)
}
