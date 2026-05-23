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

func main() {
	os.Exit(run())
}

func run() int {
	opts := parseFlags()
	cfg := config.Default()
	cfg.OutputFormat = opts.outputFormat
	cfg.OutputPath = opts.outputPath
	cfg.SameOrigin = !opts.allowCrossOrigin
	cfg.LogLevel = opts.logLevel
	cfg.WordlistPath = opts.wordlistPath
	cfg.DisableBuiltinWordlist = opts.noBuiltinWordlist
	logger.SetLevel(cfg.LogLevel)

	if err := applyHeaders(&cfg, opts.headers); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	if opts.cookie != "" {
		cfg.Cookies = opts.cookie
	}
	if opts.maxResources > 0 {
		cfg.MaxResources = opts.maxResources
	}
	if opts.depth > 0 {
		cfg.MaxDepth = opts.depth
	}
	if opts.concurrency > 0 {
		cfg.RequestConcurrency = opts.concurrency
	}

	if opts.url == "" {
		fmt.Fprintln(os.Stderr, "missing required flag: -u")
		return 1
	}
	if opts.outputFormat != "table" && opts.outputFormat != "json" {
		fmt.Fprintln(os.Stderr, "unsupported format, use table or json")
		return 1
	}
	if !isValidLogLevel(cfg.LogLevel) {
		fmt.Fprintln(os.Stderr, "unsupported log level, use silent, info, warn, or debug")
		return 1
	}

	logger.Info("APIExtractor v0.1.2 module 3 MVP is running")
	logger.Info(fmt.Sprintf("Target URL: %s", opts.url))
	logger.Debug(fmt.Sprintf("Config: same_origin=%t concurrency=%d format=%s", cfg.SameOrigin, cfg.RequestConcurrency, cfg.OutputFormat))

	result := core.Run(opts.url, cfg)
	if err := exporter.ExportResults(result, cfg); err != nil {
		logger.Error(fmt.Sprintf("failed to export results: %v", err))
		return 4
	}
	return 0
}

type cliOptions struct {
	url               string
	outputFormat      string
	outputPath        string
	allowCrossOrigin  bool
	headers           headerFlags
	cookie            string
	logLevel          string
	wordlistPath      string
	noBuiltinWordlist bool
	maxResources      int
	depth             int
	concurrency       int
}

func parseFlags() cliOptions {
	var opts cliOptions

	flag.StringVar(&opts.url, "u", "", "Target page URL")
	flag.StringVar(&opts.outputFormat, "format", "table", "Output format: table or json")
	flag.StringVar(&opts.outputPath, "o", "", "Optional output file path")
	flag.BoolVar(&opts.allowCrossOrigin, "allow-cross-origin", false, "Keep candidates from other origins")
	flag.Var(&opts.headers, "header", "Custom header in Key: Value format, can be repeated")
	flag.StringVar(&opts.cookie, "cookie", "", "Optional cookie header value")
	flag.StringVar(&opts.logLevel, "log-level", "info", "Log level: silent, info, warn, debug")
	flag.StringVar(&opts.wordlistPath, "wordlist", "", "Optional local wordlist file path")
	flag.BoolVar(&opts.noBuiltinWordlist, "no-builtin-wordlist", false, "Disable builtin wordlist entries")
	flag.IntVar(&opts.depth, "depth", 2, "Maximum recursion depth")
	flag.IntVar(&opts.maxResources, "max-resources", 200, "Maximum directory-scan resources")
	flag.IntVar(&opts.concurrency, "concurrency", 10, "Maximum concurrent requests")
	flag.Parse()

	return opts
}

type headerFlags []string

func (h *headerFlags) String() string {
	return strings.Join(*h, ",")
}

func (h *headerFlags) Set(value string) error {
	*h = append(*h, value)
	return nil
}

func applyHeaders(cfg *config.Config, headers []string) error {
	for _, item := range headers {
		parts := strings.SplitN(item, ":", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid header format: %s", item)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key == "" {
			return fmt.Errorf("invalid header key: %s", item)
		}
		cfg.DefaultHeaders[key] = value
	}
	return nil
}

func isValidLogLevel(level string) bool {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "silent", "info", "warn", "debug":
		return true
	default:
		return false
	}
}
