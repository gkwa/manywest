package main

import (
	"os"

	"github.com/gkwa/manywest"
)

func main() {
	code := manywest.Execute()
	os.Exit(code)
}
