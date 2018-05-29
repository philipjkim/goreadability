package readability

import (
	"io/ioutil"
	"log"
	"os"
)

var logger = log.New(ioutil.Discard, "[readability] ", log.LstdFlags)

func init() {
	if getOrDefault("DEBUG", "false") == "true" {
		Debug()
	}
}

// Debug enables debug logging of the operations done by the library.
// If called, lots of information will be print to stdout.
func Debug() {
	logger = log.New(os.Stdout, "[readability] ", log.LstdFlags)
}
