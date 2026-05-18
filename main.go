package main

import (
	"flag"
	"fmt"
	"os"

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
	url              string
	outputFormat     string
	outputPath       string
	allowCrossOrigin bool
}

// parseFlags 读取命令行参数并转换为内部选项结构。
func parseFlags() cliOptions {
	var opts cliOptions

	flag.StringVar(&opts.url, "u", "", "Target page URL")
	flag.StringVar(&opts.outputFormat, "format", "table", "Output format: table or json")
	flag.StringVar(&opts.outputPath, "o", "", "Optional output file path")
	flag.BoolVar(&opts.allowCrossOrigin, "allow-cross-origin", false, "Keep candidates from other origins")
	flag.Parse()

	return opts
}
