package core

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"apiextractor/internal/config"
	"apiextractor/internal/model"
	"apiextractor/internal/urlutil"
)

var builtinMiniWordlist = []string{
	"/api",
	"/api/v1",
	"/api/v2",
	"/api/ping",
	"/api/version",
	"/graphql",
	"/swagger",
	"/openapi.json",
	"/robots.txt",
	"/sitemap.xml",
	"/admin",
	"/admin/login",
	"/backend",
	"/console",
}

// LoadWordlists loads builtin and optional user wordlists, then normalizes and deduplicates them.
func LoadWordlists(targetURL string, cfg config.Config) ([]string, []model.WordlistMeta, error) {
	all := make([]string, 0)
	metas := make([]model.WordlistMeta, 0, 2)

	if !cfg.DisableBuiltinWordlist {
		items := CleanWordlistEntries(builtinMiniWordlist)
		all = append(all, items...)
		metas = append(metas, model.WordlistMeta{
			WordlistName:    "builtin-mini",
			WordlistVersion: "0.1.2",
			SourceType:      "builtin",
			UpdatedAt:       "2026-05-22",
			EntryCount:      len(items),
			Category:        "directory-scan",
			Maintainer:      "APIExtractor",
			SHA256:          hashEntries(items),
		})
	}

	if cfg.WordlistPath != "" {
		items, meta, err := loadWordlistFile(cfg.WordlistPath)
		if err != nil {
			return nil, nil, err
		}
		all = append(all, items...)
		metas = append(metas, meta)
	}

	all = CleanWordlistEntries(all)
	all = BuildScanTargets(targetURL, all)
	return all, metas, nil
}

// CleanWordlistEntries trims comments, blanks, duplicates, and normalizes slashes.
func CleanWordlistEntries(items []string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(items))
	for _, item := range items {
		cleaned := strings.TrimSpace(item)
		if cleaned == "" {
			continue
		}
		if idx := strings.Index(cleaned, "#"); idx == 0 {
			continue
		}
		if idx := strings.Index(cleaned, "#"); idx > 0 {
			cleaned = strings.TrimSpace(cleaned[:idx])
		}
		if cleaned == "" {
			continue
		}
		cleaned = strings.ReplaceAll(cleaned, "\\", "/")
		if !strings.HasPrefix(cleaned, "/") && !strings.HasPrefix(cleaned, "http://") && !strings.HasPrefix(cleaned, "https://") {
			cleaned = "/" + cleaned
		}
		if _, exists := seen[cleaned]; exists {
			continue
		}
		seen[cleaned] = struct{}{}
		out = append(out, cleaned)
	}
	sort.Strings(out)
	return out
}

// BuildScanTargets resolves wordlist entries against the target origin.
func BuildScanTargets(targetURL string, entries []string) []string {
	origin := urlutil.Origin(targetURL)
	out := make([]string, 0, len(entries))
	seen := make(map[string]struct{})
	for _, item := range entries {
		resolved := item
		if strings.HasPrefix(item, "/") {
			resolved = origin + item
		}
		if _, exists := seen[resolved]; exists {
			continue
		}
		seen[resolved] = struct{}{}
		out = append(out, resolved)
	}
	sort.Strings(out)
	return out
}

func loadWordlistFile(path string) ([]string, model.WordlistMeta, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, model.WordlistMeta{}, err
	}
	defer file.Close()

	items := make([]string, 0)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		items = append(items, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, model.WordlistMeta{}, err
	}

	items = CleanWordlistEntries(items)
	meta := model.WordlistMeta{
		WordlistName: filepath.Base(path),
		SourceType:   "user_file",
		SourceURL:    path,
		EntryCount:   len(items),
		Category:     "directory-scan",
		SHA256:       hashEntries(items),
	}
	return items, meta, nil
}

func hashEntries(items []string) string {
	sum := sha256.Sum256([]byte(strings.Join(items, "\n")))
	return hex.EncodeToString(sum[:])
}
