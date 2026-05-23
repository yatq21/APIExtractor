package config

import "time"

type Config struct {
	Timeout                  time.Duration
	OutputFormat             string
	OutputPath               string
	SameOrigin               bool
	RequestConcurrency       int
	LogLevel                 string
	DefaultHeaders           map[string]string
	Cookies                  string
	WordlistPath             string
	DisableBuiltinWordlist   bool
	MaxResources             int
	MaxDepth                 int
	StaticExtension          map[string]struct{}
	MaxSourceFiles           int
	MaxResponsePreview       int
	MaxResponseScanBytes     int64
	LargeJSONThreshold       int
	FollowSameOriginRedirect bool
}

// Default returns the baseline scan configuration.
func Default() Config {
	return Config{
		Timeout:                  10 * time.Second,
		OutputFormat:             "table",
		SameOrigin:               true,
		RequestConcurrency:       6,
		LogLevel:                 "info",
		MaxResources:             200,
		MaxDepth:                 2,
		MaxSourceFiles:           40,
		MaxResponsePreview:       2 << 10,
		MaxResponseScanBytes:     2 << 20,
		LargeJSONThreshold:       100 << 10,
		FollowSameOriginRedirect: true,
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
