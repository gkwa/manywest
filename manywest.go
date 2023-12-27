package manywest

import (
	"flag"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/h2non/filetype"
)

var excludeDirs = map[string]bool{
	".git":        true,
	"__pycache__": true,
}

type Options struct {
	LogFormat string
	LogLevel  string
}

func Execute() int {
	options := parseArgs()

	logger, err := getLogger(options.LogLevel, options.LogFormat)
	if err != nil {
		slog.Error("getLogger", "error", err)
		return 1
	}

	slog.SetDefault(logger)

	err = run(options)
	if err != nil {
		slog.Error("run failed", "error", err)
		return 1
	}
	return 0
}

func parseArgs() Options {
	options := Options{}

	flag.StringVar(&options.LogLevel, "log-level", "info", "Log level (debug, info, warn, error), default: info")
	flag.StringVar(&options.LogFormat, "log-format", "text", "Log format (text or json)")

	flag.Parse()

	return options
}

const templateScript = `#!/usr/bin/env bash

# set -e

tmp=$(mktemp -d {{.Cwd}}.XXXXX)

if [ -z "${tmp+x}" ] || [ -z "$tmp" ]; then
    echo "error: $tmp is not set or is an empty string."
    exit 1
fi

if ! command -v txtar-c >/dev/null; then
    echo go install github.com/rogpeppe/go-internal/cmd/txtar-c@latest
	exit 1
fi

{{range .Files}}# ls '{{.}}' >>$tmp/filelist.txt
{{end}}

tar -cf $tmp/{{.Cwd}}.tar -T $tmp/filelist.txt
mkdir -p $tmp/{{.Cwd}}
tar xf $tmp/{{.Cwd}}.tar -C $tmp/{{.Cwd}}
rg --files $tmp/{{.Cwd}}
txtar-c $tmp/{{.Cwd}} | pbcopy

rm -rf $tmp
`

func isFileText(filename string) (bool, error) {
	file, err := os.Open(filename)
	if err != nil {
		return false, err
	}
	defer file.Close()

	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil && err != io.EOF {
		return false, err
	}

	kind, _ := filetype.Match(buffer)
	return kind == filetype.Unknown || kind.MIME.Type == "text", nil
}

func run(options Options) error {
	filename, err := filepath.Abs("make_txtar.sh")
	if err != nil {
		panic(err)
	}
	_, err = os.Stat(filename)
	if err == nil {
		slog.Warn("file exists, quitting early to prevent overwriting", "file", filename)
		return nil
	}

	fileList, err := recurseDirectory(".")
	if err != nil {
		slog.Error("error:", "error", err)
		return err
	}

	const MAX_FILE_COUNT = 100

	if len(fileList) > MAX_FILE_COUNT {
		slog.Error("error: Number of files is greater than 20.", "fileCount", len(fileList))
		return err
	}

	tmpl, err := template.New("script").Parse(templateScript)
	if err != nil {
		slog.Error("error:", "error", err)
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		slog.Error("error:", "error", err)
		return err
	}

	cwdName := filepath.Base(cwd)

	data := struct {
		Files []string
		Cwd   string
	}{
		Files: fileList,
		Cwd:   cwdName,
	}

	var scriptBuilder strings.Builder
	err = tmpl.Execute(&scriptBuilder, data)
	if err != nil {
		slog.Error("error:", "error", err)
		return err
	}

	scriptFileName := "make_txtar.sh"
	err = os.WriteFile(scriptFileName, []byte(scriptBuilder.String()), 0o755)
	if err != nil {
		slog.Error("error:", "error", err)
		return err
	}

	slog.Info("Script created successfully", "script", scriptFileName)
	return nil
}

func recurseDirectory(root string) ([]string, error) {
	var fileList []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && excludeDirs[info.Name()] {
			return filepath.SkipDir
		}
		if info.IsDir() {
			return nil
		}
		if isExcludedFile(info.Name()) {
			return nil
		}

		// Check if the file is text and skip if it is not
		isText, err := isFileText(path)
		if err != nil {
			slog.Error("Error checking if file is text", "file", path, "error", err)
			return err
		}
		if isText {
			slog.Debug("file is text file", "file", path)
		}

		if !isText {
			slog.Info("Skipping non-text file", "file", path)
			return nil
		}

		fileList = append(fileList, path)
		return nil
	})

	return fileList, err
}

func isExcludedFile(fileName string) bool {
	excludedFiles := map[string]bool{
		// Add more excluded files as needed
	}

	return excludedFiles[fileName]
}
