package manywest

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

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
    echo "Error: $tmp is not set or is an empty string."
    exit 1
fi

if ! command -v txtar-c >/dev/null; then
    echo go install github.com/rogpeppe/go-internal/cmd/txtar-c@latest
	exit 1
fi

if ! command -v rg >/dev/null; then
    echo brew insall ripgrep
	exit 1
fi

{
    rg --files . \
        | grep -vE $tmp'/filelist.txt$' \
        | grep -vE 'make_txtar.sh$' \
        {{range .Files}}| grep -vE '{{.}}$' \
        {{end}}
} | tee $tmp/filelist.txt

tar -cf $tmp/{{.Cwd}}.tar -T $tmp/filelist.txt
mkdir -p $tmp/{{.Cwd}}
tar xf $tmp/{{.Cwd}}.tar -C $tmp/{{.Cwd}}
rg --files $tmp/{{.Cwd}}
txtar-c $tmp/{{.Cwd}} | pbcopy

rm -rf $tmp
`

func run(options Options) error {
	filename := "make_txtar.sh"
	_, err := os.Stat(filename)
	if err == nil {
		slog.Info("file exists, quiting early to prevent overwriting", "file", filename)
		return nil
	}

	fileList, err := recurseDirectory(".")
	if err != nil {
		slog.Error("Error:", "error", err)
		return err
	}

	if len(fileList) > 20 {
		slog.Error("Error: Number of files is greater than 20.", "fileCount", len(fileList))
		return err
	}

	tmpl, err := template.New("script").Parse(templateScript)
	if err != nil {
		slog.Error("Error:", "error", err)
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		slog.Error("Error:", "error", err)
		return err
	}

	// convert 'manywest' to regex 'manywest$'
	cwdName := filepath.Base(cwd)
	matchString := cwdName
	replaceString := fmt.Sprintf("%s$", cwdName)
	for i, s := range fileList {
		if s == matchString {
			fileList[i] = replaceString
		}
	}

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
		slog.Error("Error:", "error", err)
		return err
	}

	scriptFileName := "make_txtar.sh"
	err = os.WriteFile(scriptFileName, []byte(scriptBuilder.String()), 0o755)
	if err != nil {
		slog.Error("Error:", "error", err)
		return err
	}

	slog.Debug("Script created successfully", "script", scriptFileName)
	return nil
}

func recurseDirectory(root string) ([]string, error) {
	var fileList []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}
		if info.IsDir() {
			return nil
		}
		if isExcludedFile(info.Name()) {
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
