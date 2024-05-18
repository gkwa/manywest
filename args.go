package manywest

import (
	"fmt"
	"os"

	"github.com/jessevdk/go-flags"
)

type Options struct {
	LogFormat           string   `long:"log-format" default:"text" description:"Log format (text or json)"`
	LogLevel            string   `long:"log-level" default:"info" description:"Log level (debug, info, warn, error)"`
	ForceOverwrite      bool     `long:"force" short:"f" description:"Force overwrite pre-existing make_txtar.sh"`
	ExcludeDirs         []string `long:"ignore-dirs" short:"i" description:"Ignore directories"`
	MaxFileCount        int      `long:"maxfiles" default:"100" description:"Maximum number of files to include in txtar archive"`
	IncludeInstructions bool     `long:"include-instructions" short:"s" description:"Include instructions into txtar archive"`
	Version             bool     `long:"version" description:"Display verison"`
}

func parseArgs() Options {
	var options Options

	parser := flags.NewParser(&options, flags.Default)
	_, err := parser.Parse()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	return options
}
