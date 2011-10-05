
package main

import (
	"replican/blocks"
	"os"
	"fmt"
)

func main() {
	if (len(os.Args) < 2) {
		fmt.Printf("Usage: %s <dir>\n", os.Args[0])
		return
	}
	
	blocks.IndexDir(os.Args[1])
}

