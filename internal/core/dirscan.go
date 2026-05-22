package core

import (
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"sync"

	"apiextractor/internal/config"
	"apiextractor/internal/model"
)

// ScanDirectoryResources performs bounded concurrent checks for wordlist-driven resource discovery.
func ScanDirectoryResources(targets []string, cfg config.Config) ([]model.ResourceRecord, []string) {
	if len(targets) == 0 {
		return nil, nil
	}
	maxResources := cfg.MaxResources
	if maxResources <= 0 {
		maxResources = 200
	}
	if len(targets) > maxResources {
		targets = targets[:maxResources]
	}

	results := make([]model.ResourceRecord, len(targets))
	sem := make(chan struct{}, max(1, cfg.RequestConcurrency))
	var wg sync.WaitGroup

	for i, target := range targets {
		wg.Add(1)
		go func(index int, raw string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			results[index] = probeResource(index+1, raw, cfg)
		}(i, target)
	}
	wg.Wait()

	out := make([]model.ResourceRecord, 0, len(results))
	budgetHits := make([]string, 0)
	for _, item := range results {
		if item.URL == "" {
			continue
		}
		if item.FetchError != "" || item.Type == "" {
			continue
		}
		out = append(out, item)
	}
	if len(targets) == maxResources {
		budgetHits = append(budgetHits, "max_resources_reached")
	}
	return out, budgetHits
}

func probeResource(index int, rawURL string, cfg config.Config) model.ResourceRecord {
	record := model.ResourceRecord{
		ResourceID:    resourceID(index),
		URL:           rawURL,
		Path:          scannedResourcePath(rawURL),
		Category:      "directory-scan",
		Source:        "wordlist-scan",
		SameOrigin:    true,
		ShouldAnalyze: true,
	}

	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		record.ErrorType = classifyError(err)
		record.FetchError = truncateError(err.Error())
		return record
	}
	for key, value := range cfg.DefaultHeaders {
		req.Header.Set(key, value)
	}
	if cfg.Cookies != "" {
		req.Header.Set("Cookie", cfg.Cookies)
	}

	client := &http.Client{Timeout: cfg.Timeout}
	resp, err := client.Do(req)
	if err != nil {
		record.ErrorType = classifyError(err)
		record.FetchError = truncateError(err.Error())
		return record
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		record.FetchError = http.StatusText(resp.StatusCode)
		record.ErrorType = "http_" + strings.ToLower(strings.ReplaceAll(http.StatusText(resp.StatusCode), " ", "_"))
		return record
	}
	record.Type = classifyResourceType(rawURL, resp.Header.Get("Content-Type"))
	record.ContentType = resp.Header.Get("Content-Type")
	record.Tags = discoveryTags(record.Type, rawURL)
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, int64(cfg.MaxResponsePreview)))
	if readErr == nil {
		record.BodyPreview = strings.TrimSpace(string(body))
	}
	return record
}

func classifyResourceType(rawURL string, contentType string) string {
	lowerURL := strings.ToLower(rawURL)
	lowerType := strings.ToLower(contentType)

	switch {
	case strings.Contains(lowerURL, "robots.txt"):
		return "robots"
	case strings.Contains(lowerURL, "sitemap") && strings.HasSuffix(lowerURL, ".xml"):
		return "sitemap"
	case strings.Contains(lowerURL, "manifest") && strings.HasSuffix(lowerURL, ".json"):
		return "manifest"
	}

	switch {
	case strings.Contains(lowerType, "html"):
		return "html"
	case strings.Contains(lowerType, "javascript"):
		return "javascript"
	case strings.Contains(lowerType, "json"):
		return "json"
	case strings.Contains(lowerType, "xml"):
		return "xml"
	}

	switch strings.ToLower(filepath.Ext(rawURL)) {
	case ".js", ".mjs":
		return "javascript"
	case ".map":
		return "sourcemap"
	case ".json":
		return "json"
	case ".xml":
		return "xml"
	case ".txt":
		return "text"
	default:
		return "html"
	}
}

func max(left int, right int) int {
	if left > right {
		return left
	}
	return right
}

func scannedResourcePath(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	if parsed.EscapedPath() == "" {
		return "/"
	}
	return parsed.EscapedPath()
}

func discoveryTags(resourceType string, rawURL string) []string {
	tags := make([]string, 0, 3)
	switch resourceType {
	case "robots", "sitemap", "manifest":
		tags = append(tags, "discovery-hint")
	case "api_doc":
		tags = append(tags, "api-doc")
	}
	lowerURL := strings.ToLower(rawURL)
	if strings.Contains(lowerURL, "admin") || strings.Contains(lowerURL, "backend") || strings.Contains(lowerURL, "console") {
		tags = append(tags, "admin-path")
	}
	return tags
}
