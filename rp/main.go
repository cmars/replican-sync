package main

import (
	"fmt"
	"os"

	"github.com/cmars/replican-sync/replican"
)

func main() {
	argv := os.Args[1:]
	cmd := argv[0]
	
	switch cmd {
	case "index":
		if len(argv[1:]) < 1 {
			fmt.Fprintf(os.Stderr, "Missing argument: path\n", cmd)
			usage()
		}
		index(argv[1])
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		usage()
	}
	os.Exit(0)
}

func usage() {
	fmt.Fprintf(os.Stderr, "FAIL")
	os.Exit(1)
}

func index(path string) {
	index, err := replican.WriteIndex(path)
	defer index.Close()
	if err != nil {
		panic(err)
	}
}
