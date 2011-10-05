package main
 
import (
    "fmt"
    "path/filepath"
    "os"
)

type MyVisitor struct{
	
} // a type to satisfy filepath.Visitor interface
 
func (MyVisitor) VisitDir(string, *os.FileInfo) bool {
	
    return true
}

func (MyVisitor) VisitFile(fp string, _ *os.FileInfo) {
//    if filepath.Ext(fp) == ".mp3" {
        fmt.Println(fp)
//    }
}
 
func main() {
    filepath.Walk(os.Args[1], MyVisitor{}, nil)
}

