
package main

import "fmt"
import "strconv"
import "os"

func fib(n uint64) uint64 {
	var i uint64
	var j uint64
	for i, j = 1, 1; n > 0; i, j, n = j, i+j, n-1 {}
	return j
}

func main() {
	n, _ := strconv.Atoui64(os.Args[1])
	fn := fib(n)
	fmt.Printf("fib %d = %d\n", n, fn)
}

