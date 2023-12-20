package main

import (
	"os"

	"github.com/taylormonacelli/manywest"
)

func main() {
	code := manywest.Execute()
	os.Exit(code)
}
