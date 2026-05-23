package config

import "time"

type Config struct {
	Timeout                  time.Duration
	OutputFormat             string
	OutputPath               string
	SameOrigin               bool
	DefaultHeaders           map[string]string
	StaticExtension          map[string]struct{}
	MaxSourceFiles           int
	EnableDirectoryScan      bool
	UseBuiltinDictionary     bool
	DictionaryPaths          []string
	MaxDirectoryScanEntries  int
	DirectoryScanConcurrency int
	EnableSoft404Detection   bool
}

// Default 返回命令行扫描使用的默认配置。
func Default() Config {
	return Config{
		Timeout:                  10 * time.Second,
		OutputFormat:             "table",
		SameOrigin:               true,
		MaxSourceFiles:           40,
		EnableDirectoryScan:      true,
		UseBuiltinDictionary:     true,
		MaxDirectoryScanEntries:  80,
		DirectoryScanConcurrency: 10,
		EnableSoft404Detection:   true,
		DefaultHeaders: map[string]string{
			"User-Agent": "Mozilla/5.0 APIExtractor-Go",
		},
		StaticExtension: map[string]struct{}{
			".css":   {},
			".gif":   {},
			".ico":   {},
			".jpeg":  {},
			".jpg":   {},
			".png":   {},
			".svg":   {},
			".webp":  {},
			".woff":  {},
			".woff2": {},
		},
	}
}
