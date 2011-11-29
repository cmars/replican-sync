package track

import (
	"io/ioutil"
	"log"
	"os"
)

func StdLog() *log.Logger {
	return log.New(os.Stderr, "", log.LstdFlags|log.Lshortfile)
}

func NullLog() *log.Logger {
	return log.New(ioutil.Discard, "", log.LstdFlags)
}
