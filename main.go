package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"apiextractor/internal/config"
	"apiextractor/internal/core"
	"apiextractor/internal/exporter"
	"apiextractor/internal/logger"
)

// main 解析命令行参数，构建运行配置，并启动一次扫描。
func main() {
	opts := parseFlags()
	cfg := config.Default()
	cfg.OutputFormat = opts.outputFormat
	cfg.OutputPath = opts.outputPath
	cfg.SameOrigin = !opts.allowCrossOrigin
	cfg.EnableDirectoryScan = !opts.disableDirectoryScan
	cfg.UseBuiltinDictionary = !opts.disableBuiltinDictionary
	cfg.DictionaryPaths = opts.dictionaryPaths
	cfg.DirectoryScanConcurrency = opts.concurrency
	cfg.EnableSoft404Detection = !opts.disableSoft404Detection

	if opts.url == "" {
		fmt.Fprintln(os.Stderr, "missing required flag: -u")
		os.Exit(1)
	}

	logger.Info("APIExtractor Go skeleton is ready.")
	logger.Info(fmt.Sprintf("Target URL: %s", opts.url))

	result := core.Run(opts.url, cfg)
	if err := exporter.ExportResults(result, cfg); err != nil {
		logger.Error(fmt.Sprintf("failed to export results: %v", err))
		os.Exit(1)
	}
}

type cliOptions struct {
	url                      string
	outputFormat             string
	outputPath               string
	allowCrossOrigin         bool
	disableDirectoryScan     bool
	disableBuiltinDictionary bool
	disableSoft404Detection  bool
	concurrency              int
	dictionaryPathList       string
	dictionaryPaths          []string
}

// parseFlags 读取命令行参数并转换为内部选项结构。
func parseFlags() cliOptions {
	var opts cliOptions

	flag.StringVar(&opts.url, "u", "", "Target page URL")
	flag.StringVar(&opts.outputFormat, "format", "table", "Output format: table or json")
	flag.StringVar(&opts.outputPath, "o", "", "Optional output file path")
	flag.BoolVar(&opts.allowCrossOrigin, "allow-cross-origin", false, "Keep candidates from other origins")
	flag.BoolVar(&opts.disableDirectoryScan, "no-dir-scan", false, "Disable dictionary based directory/resource discovery")
	flag.BoolVar(&opts.disableBuiltinDictionary, "no-builtin-dict", false, "Disable builtin dictionary entries")
	flag.BoolVar(&opts.disableSoft404Detection, "no-soft-404", false, "Disable soft 404 baseline filtering")
	flag.IntVar(&opts.concurrency, "c", 10, "Directory scan concurrency")
	flag.IntVar(&opts.concurrency, "concurrency", 10, "Directory scan concurrency")
	flag.StringVar(&opts.dictionaryPathList, "dict", "", "Local dictionary file path, use comma to separate multiple files")
	flag.Parse()

	opts.dictionaryPaths = splitList(opts.dictionaryPathList)
	return opts
}

func splitList(raw string) []string {
	parts := strings.Split(raw, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item == "" {
			continue
		}
		items = append(items, item)
	}
	return items
}
