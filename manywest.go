package manywest

import (
	"bufio"
	_ "embed"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/gkwa/manywest/version"
	"github.com/h2non/filetype"
)

type FileEntry struct {
	Path  string
	Count int
	Type  string
}

//go:embed template.tpl
var templateScript string

func run(options Options) error {
	excludeDirs := map[string]bool{
		"__pycache__":             true,
		".git":                    true,
		".tox":                    true,
		".ruff_cache":             true,
		".pytest_cache":           true,
		".terraform":              true,
		".timestamps":             true,
		".venv":                   true,
		"gpt_instructions_XXYYBB": true,
		"node_modules":            true,
		"target/debug":            true,
	}
	for _, dir := range options.ExcludeDirs {
		excludeDirs[dir] = true
	}

	filename, _ := filepath.Abs("make_txtar.sh")
	_, err := os.Stat(filename)
	if err == nil && !options.ForceOverwrite {
		slog.Warn("file exists, quitting early to prevent overwriting", "file", filename)
		return nil
	}

	fileList, err := recurseDirectory(".", excludeDirs)
	if err != nil {
		slog.Error("error:", "error", err)
		return err
	}

	if len(fileList) > options.MaxFileCount {
		slog.Error("error: Number of files is greater than limit", "limit", options.MaxFileCount, "fileCount", len(fileList))
		return err
	}

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
		Files               []FileEntry
		Cwd                 string
		IncludeInstructions bool
	}{
		Files:               fileEntries,
		Cwd:                 cwdName,
		IncludeInstructions: options.IncludeInstructions,
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

func recurseDirectory(root string, excludeDirs map[string]bool) ([]string, error) {
	var fileList []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			for excl := range excludeDirs {
				if strings.Contains(strings.ToLower(path), strings.ToLower(excl)) {
					return filepath.SkipDir
				}
			}
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

func Execute() int {
	options := parseArgs()

	if options.Version {
		fmt.Println(version.GetBuildInfo())
		os.Exit(0)
	}

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
