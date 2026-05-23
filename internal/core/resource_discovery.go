package core

import (
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"apiextractor/internal/config"
	"apiextractor/internal/model"
)

type discoveredResource struct {
	record  model.ResourceRecord
	content string
}

// DiscoverResources probes dictionary entries under the target origin and returns reusable resource records.
func DiscoverResources(targetURL string, dictionary []string, cfg config.Config) ([]model.ResourceRecord, []model.SourceFile) {
	if !cfg.EnableDirectoryScan || len(dictionary) == 0 {
		return nil, nil
	}

	baseURL, err := url.Parse(targetURL)
	if err != nil {
		return nil, nil
	}

	resourceURLs := buildDirectoryScanURLs(dictionary, baseURL, cfg)
	return probeResourceURLs(resourceURLs, targetURL, cfg)
}

func buildDirectoryScanURLs(dictionary []string, baseURL *url.URL, cfg config.Config) []string {
	limit := directoryScanLimit(len(dictionary), cfg.MaxDirectoryScanEntries)
	seen := make(map[string]struct{}, limit)
	resourceURLs := make([]string, 0, limit)

	for _, entry := range dictionary[:limit] {
		resourceURL, ok := buildResourceURL(entry, baseURL, cfg.SameOrigin)
		if !ok {
			continue
		}
		if _, exists := seen[resourceURL]; exists {
			continue
		}
		seen[resourceURL] = struct{}{}
		resourceURLs = append(resourceURLs, resourceURL)
	}

	return resourceURLs
}

func directoryScanLimit(dictionaryCount int, configuredLimit int) int {
	if configuredLimit <= 0 {
		configuredLimit = 80
	}
	if dictionaryCount < configuredLimit {
		return dictionaryCount
	}
	return configuredLimit
}

func probeResourceURLs(resourceURLs []string, targetURL string, cfg config.Config) ([]model.ResourceRecord, []model.SourceFile) {
	records := make([]model.ResourceRecord, 0, len(resourceURLs))
	sourceFiles := make([]model.SourceFile, 0)

	for _, resourceURL := range resourceURLs {
		discovered := ProbeResource(resourceURL, targetURL, cfg, "dictionary")
		if !shouldKeepResource(discovered.record) {
			continue
		}

		records = append(records, discovered.record)
		sourceFile, ok := sourceFileFromResource(discovered)
		if ok {
			sourceFiles = append(sourceFiles, sourceFile)
		}
	}

	return records, sourceFiles
}

// ProbeResource sends one GET request and records metadata needed by resource discovery output.
func ProbeResource(rawURL string, baseURL string, cfg config.Config, discoverSource string) discoveredResource {
	start := time.Now()
	record := newResourceRecord(rawURL, baseURL, discoverSource)
	req, err := newResourceRequest(rawURL, cfg)
	if err != nil {
		record.FetchError = err.Error()
		return discoveredResource{record: record}
	}

	client := &http.Client{Timeout: cfg.Timeout}
	resp, err := client.Do(req)
	record.DurationMS = time.Since(start).Milliseconds()
	if err != nil {
		record.FetchError = err.Error()
		return discoveredResource{record: record}
	}
	defer resp.Body.Close()

	body, readErr := readLimitedResourceBody(resp.Body)
	if readErr != nil {
		record.FetchError = readErr.Error()
		return discoveredResource{record: record}
	}

	fillResourceResponse(&record, resp, body)
	return discoveredResource{
		record:  record,
		content: string(body),
	}
}

func newResourceRecord(rawURL string, baseURL string, discoverSource string) model.ResourceRecord {
	return model.ResourceRecord{
		URL:            rawURL,
		Method:         http.MethodGet,
		DiscoverSource: discoverSource,
		ResourceType:   "unknown",
		SameOrigin:     isSameOrigin(rawURL, baseURL),
	}
}

func newResourceRequest(rawURL string, cfg config.Config) (*http.Request, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	for key, value := range cfg.DefaultHeaders {
		req.Header.Set(key, value)
	}
	return req, nil
}

func readLimitedResourceBody(body io.Reader) ([]byte, error) {
	return io.ReadAll(io.LimitReader(body, 2<<20))
}

func fillResourceResponse(record *model.ResourceRecord, resp *http.Response, body []byte) {
	record.FinalURL = resp.Request.URL.String()
	record.StatusCode = resp.StatusCode
	record.ContentType = resp.Header.Get("Content-Type")
	record.ContentLength = len(body)
	record.ResourceType = DetectResourceType(record.FinalURL, record.ContentType)
	record.ShouldAnalyze = shouldAnalyzeResource(record.ResourceType, record.ContentType)
}

func sourceFileFromResource(discovered discoveredResource) (model.SourceFile, bool) {
	if !discovered.record.ShouldAnalyze || discovered.content == "" {
		return model.SourceFile{}, false
	}

	return model.SourceFile{
		URL:        effectiveResourceURL(discovered.record),
		SourceType: discovered.record.ResourceType,
		Content:    discovered.content,
	}, true
}

func buildResourceURL(entry string, baseURL *url.URL, sameOrigin bool) (string, bool) {
	parsed, err := url.Parse(entry)
	if err != nil {
		return "", false
	}
	resolved := baseURL.ResolveReference(parsed)
	if resolved.Scheme != "http" && resolved.Scheme != "https" {
		return "", false
	}
	if sameOrigin && !sameOriginParsedURL(resolved, baseURL) {
		return "", false
	}
	resolved.Fragment = ""
	return resolved.String(), true
}

func shouldKeepResource(record model.ResourceRecord) bool {
	if record.FetchError != "" {
		return false
	}
	if record.StatusCode >= 200 && record.StatusCode < 400 {
		return true
	}
	if record.StatusCode == http.StatusUnauthorized || record.StatusCode == http.StatusForbidden {
		return true
	}
	return false
}

func shouldAnalyzeResource(resourceType string, contentType string) bool {
	contentType = strings.ToLower(contentType)
	switch resourceType {
	case "html", "javascript", "chunk_js", "source_map", "json", "text", "api_doc":
		return true
	}
	return strings.Contains(contentType, "json") || strings.Contains(contentType, "text")
}

// DetectResourceType classifies a response using URL path first, then Content-Type.
func DetectResourceType(rawURL string, contentType string) string {
	lowerPath := lowerURLPath(rawURL)
	if resourceType := detectResourceTypeByPath(lowerPath); resourceType != "" {
		return resourceType
	}

	return detectResourceTypeByContentType(contentType)
}

func lowerURLPath(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return strings.ToLower(rawURL)
	}
	return strings.ToLower(parsed.Path)
}

func detectResourceTypeByPath(lowerPath string) string {
	if strings.Contains(lowerPath, "swagger") || strings.Contains(lowerPath, "openapi") || strings.Contains(lowerPath, "api-docs") {
		return "api_doc"
	}
	ext := strings.ToLower(path.Ext(lowerPath))
	switch ext {
	case ".html", ".htm":
		return "html"
	case ".js", ".mjs":
		if strings.Contains(lowerPath, "chunk") {
			return "chunk_js"
		}
		return "javascript"
	case ".map":
		return "source_map"
	case ".json":
		return "json"
	case ".txt", ".xml":
		return "text"
	case ".css", ".gif", ".ico", ".jpeg", ".jpg", ".png", ".svg", ".webp", ".woff", ".woff2":
		return "static_resource"
	default:
		return ""
	}
}

func detectResourceTypeByContentType(contentType string) string {
	contentType = strings.ToLower(contentType)

	switch {
	case strings.Contains(contentType, "html"):
		return "html"
	case strings.Contains(contentType, "javascript") || strings.Contains(contentType, "ecmascript"):
		return "javascript"
	case strings.Contains(contentType, "json"):
		return "json"
	case strings.Contains(contentType, "text") || strings.Contains(contentType, "xml"):
		return "text"
	default:
		return "unknown"
	}
}

func isSameOrigin(leftRaw string, rightRaw string) bool {
	left, err := url.Parse(leftRaw)
	if err != nil {
		return false
	}
	right, err := url.Parse(rightRaw)
	if err != nil {
		return false
	}
	return sameOriginParsedURL(left, right)
}

func effectiveResourceURL(record model.ResourceRecord) string {
	if record.FinalURL != "" {
		return record.FinalURL
	}
	return record.URL
}

func sameOriginParsedURL(left *url.URL, right *url.URL) bool {
	return strings.EqualFold(left.Scheme, right.Scheme) && strings.EqualFold(left.Host, right.Host)
}
