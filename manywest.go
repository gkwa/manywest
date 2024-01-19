package manywest

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/h2non/filetype"
	"github.com/jessevdk/go-flags"
)

var excludeDirs = map[string]bool{
	".git":                    true,
	"__pycache__":             true,
	"node_modules":            true,
	"gpt_instructions_XXYYBB": true,
	".ruff_cache":             true,
}

type FileEntry struct {
	Path  string
	Count int
	Type  string
}

type Options struct {
	LogFormat      string `long:"log-format" default:"text" description:"Log format (text or json)"`
	LogLevel       string `long:"log-level" default:"info" description:"Log level (debug, info, warn, error)"`
	ForceOverwrite bool   `long:"force" short:"f" description:"Force overwrite pre-existing make_txtar.sh"`
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
	var options Options

	parser := flags.NewParser(&options, flags.Default)
	_, err := parser.Parse()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	return options
}

const templateScript = `#!/usr/bin/env bash
set -e

tmp=$(mktemp -d {{.Cwd}}.XXXXX)

if [ -z "${tmp+x}" ] || [ -z "$tmp" ]; then
    echo "error: $tmp is not set or is an empty string."
    exit 1
fi

if ! command -v txtar-c >/dev/null; then
    echo go install github.com/rogpeppe/go-internal/cmd/txtar-c@latest
	exit 1
fi

declare -a files=(
	{{range .Files}}# {{.Path}} # loc: {{.Count}}
	{{end}}
)
for file in "${files[@]}"; do
    echo $file
done | tee $tmp/filelist.txt

tar -cf $tmp/{{.Cwd}}.tar -T $tmp/filelist.txt
mkdir -p $tmp/{{.Cwd}}
tar xf $tmp/{{.Cwd}}.tar -C $tmp/{{.Cwd}}
rg --files $tmp/{{.Cwd}}

mkdir -p $tmp/gpt_instructions_XXYYBB

cat >$tmp/gpt_instructions_XXYYBB/1.txt <<EOF
Subject: Code Submission Guidelines in Txtar Archive Format

As we collaborate on code submissions, I would like to emphasize some guidelines for presenting your code using the txtar archive format.

Unified Code Block:
Ensure that all your code is displayed within a single code block using the txtar archive format. This helps maintain a structured and organized presentation.

Modification Verification:
If, upon review, you find that you haven't made any modifications to a specific source file since its initial state, kindly refrain from including it in the code block.

Txtar Archive Format Summary:
The txtar archive format should follow this structure:

-- cmd/main.go --
{ contents of main.go go here }
-- mypackage.go --
{ contents of mypackage.go go here }

Omitting Unchanged Files:
If a file requires no changes, please exclude it from the txtar archive. Do not include statements like // ... (unchanged) or similar indications.

Avoid Partial Listings:
Refrain from providing partial listings for unchanged files. Instead, either omit the file entirely or include its complete content without any abbreviations or explanations about unchanged portions.

Your adherence to these guidelines will greatly facilitate our collaboration and ensure a streamlined code submission process. Thank you for your attention to detail and cooperation.
EOF

{
    cat $tmp/gpt_instructions_XXYYBB/1.txt
    echo txtar archive is below
    txtar-c $tmp/{{.Cwd}}
} | pbcopy

rm -rf $tmp
`

func run(options Options) error {
	filename, _ := filepath.Abs("make_txtar.sh")
	_, err := os.Stat(filename)
	if err == nil && !options.ForceOverwrite {
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

	// Convert fileList to a list of FileEntry structs
	var fileEntries []FileEntry
	for _, path := range fileList {
		count, err := countLines(path)
		if err != nil {
			slog.Error("error counting lines in file", "file", path, "error", err)
			continue
		}

		fileType, _ := getFileType(path)

		entry := FileEntry{
			Path:  path,
			Count: count,
			Type:  fileType,
		}

		fileEntries = append(fileEntries, entry)
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
		Files []FileEntry
		Cwd   string
	}{
		Files: fileEntries,
		Cwd:   cwdName,
	}

	var scriptBuilder strings.Builder
	err = tmpl.Execute(&scriptBuilder, data)
	if err != nil {
		slog.Error("error:", "error", err)
		return err
	}

	err = os.WriteFile(filename, []byte(scriptBuilder.String()), 0o755)
	if err != nil {
		slog.Error("error:", "error", err)
		return err
	}

	slog.Info("script created successfully", "script", filename)
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
		if isExcludedFile(path) {
			return nil
		}

		fileList = append(fileList, path)
		return nil
	})

	return fileList, err
}

func isExcludedFile(fileName string) bool {
	isText, err := isFileText(fileName)
	if err != nil {
		slog.Error("error checking if file is text", "file", fileName, "error", err)
		return true
	}

	if isText {
		slog.Debug("filetype", "type", "text", "file", fileName)
		return false
	}

	slog.Debug("filetype", "type", "binary", "file", fileName)
	return true
}

func countLines(filename string) (int, error) {
	file, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	lineCount := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lineCount++
	}

	return lineCount, scanner.Err()
}

func getFileType(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil && err != io.EOF {
		return "", err
	}

	kind, _ := filetype.Match(buffer)
	return kind.MIME.Type, nil
}

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
