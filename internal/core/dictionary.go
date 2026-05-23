package core

import (
	"bufio"
	"os"
	"strings"

	"apiextractor/internal/config"
)

var builtinDictionary = []string{
	"/api",
	"/api/",
	"/api/v1",
	"/api/v2",
	"/v1",
	"/v2",
	"/rest",
	"/graphql",
	"/admin",
	"/admin/",
	"/backend",
	"/console",
	"/manage",
	"/swagger-ui.html",
	"/swagger/index.html",
	"/swagger-ui/index.html",
	"/api-docs",
	"/v2/api-docs",
	"/v3/api-docs",
	"/openapi.json",
	"/swagger.json",
	"/robots.txt",
	"/sitemap.xml",
	"/manifest.json",
	"/asset-manifest.json",
	"/static/js/main.js",
}

// LoadDictionary merges builtin and local dictionary entries, then normalizes them.
func LoadDictionary(cfg config.Config) ([]string, []error) {
	rawItems := loadBuiltinDictionary(cfg.UseBuiltinDictionary)
	localItems, errs := loadLocalDictionaries(cfg.DictionaryPaths)
	rawItems = append(rawItems, localItems...)

	return CleanDictionary(rawItems), errs
}

func loadBuiltinDictionary(enabled bool) []string {
	if !enabled {
		return nil
	}

	items := make([]string, 0, len(builtinDictionary))
	items = append(items, builtinDictionary...)
	return items
}

func loadLocalDictionaries(filePaths []string) ([]string, []error) {
	allItems := make([]string, 0)
	var errs []error
	for _, filePath := range filePaths {
		items, err := readDictionaryFile(filePath)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		allItems = append(allItems, items...)
	}

	return allItems, errs
}

func readDictionaryFile(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	items := make([]string, 0)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		items = append(items, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

// CleanDictionary trims comments and duplicates while preserving first-seen order.
func CleanDictionary(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	cleaned := make([]string, 0, len(items))

	for _, item := range items {
		item = cleanDictionaryEntry(item)
		if item == "" {
			continue
		}
		if _, exists := seen[item]; exists {
			continue
		}
		seen[item] = struct{}{}
		cleaned = append(cleaned, item)
	}

	return cleaned
}

func cleanDictionaryEntry(raw string) string {
	item := strings.TrimSpace(raw)
	if item == "" || strings.HasPrefix(item, "#") {
		return ""
	}
	if idx := strings.Index(item, "#"); idx >= 0 {
		item = strings.TrimSpace(item[:idx])
	}
	if item == "" {
		return ""
	}
	if strings.Contains(item, "://") || strings.HasPrefix(item, "//") {
		return item
	}
	if !strings.HasPrefix(item, "/") {
		item = "/" + item
	}
	for strings.Contains(item, "//") {
		item = strings.ReplaceAll(item, "//", "/")
	}
	return item
}
