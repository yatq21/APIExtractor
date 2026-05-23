package core

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"io"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
	"sync"
	"time"

	"apiextractor/internal/config"
	"apiextractor/internal/model"
)

type discoveredResource struct {
	record  model.ResourceRecord
	content string
}

type resourceScanJob struct {
	index int
	url   string
}

type resourceScanResult struct {
	index     int
	discovery discoveredResource
}

type soft404Baseline struct {
	enabled       bool
	statusCode    int
	contentLength int
	title         string
	bodyHash      string
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
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout*time.Duration(maxInt(1, len(resourceURLs))))
	defer cancel()

	baseline := buildSoft404Baseline(ctx, baseURL, targetURL, cfg)
	return probeResourceURLs(ctx, resourceURLs, targetURL, cfg, baseline)
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

func probeResourceURLs(ctx context.Context, resourceURLs []string, targetURL string, cfg config.Config, baseline soft404Baseline) ([]model.ResourceRecord, []model.SourceFile) {
	results := runResourceWorkers(ctx, resourceURLs, targetURL, cfg)
	sort.SliceStable(results, func(i, j int) bool {
		return results[i].index < results[j].index
	})

	records := make([]model.ResourceRecord, 0, len(resourceURLs))
	sourceFiles := make([]model.SourceFile, 0)

	for _, result := range results {
		discovered := result.discovery
		if isSoft404(discovered, baseline) || !shouldKeepResource(discovered.record) {
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

// runResourceWorkers keeps directory probing bounded by DirectoryScanConcurrency.
func runResourceWorkers(ctx context.Context, resourceURLs []string, targetURL string, cfg config.Config) []resourceScanResult {
	workerCount := directoryScanConcurrency(cfg.DirectoryScanConcurrency, len(resourceURLs))
	jobs := make(chan resourceScanJob)
	results := make(chan resourceScanResult)
	var wg sync.WaitGroup

	for workerID := 0; workerID < workerCount; workerID++ {
		wg.Add(1)
		go resourceWorker(ctx, &wg, jobs, results, targetURL, cfg)
	}

	go produceResourceJobs(ctx, resourceURLs, jobs)
	go closeResourceResults(results, &wg)

	return collectResourceResults(results)
}

func directoryScanConcurrency(configuredConcurrency int, jobCount int) int {
	if jobCount <= 0 {
		return 0
	}
	if configuredConcurrency <= 0 {
		configuredConcurrency = 10
	}
	if configuredConcurrency > jobCount {
		return jobCount
	}
	return configuredConcurrency
}

func produceResourceJobs(ctx context.Context, resourceURLs []string, jobs chan<- resourceScanJob) {
	defer close(jobs)
	for index, resourceURL := range resourceURLs {
		select {
		case <-ctx.Done():
			return
		case jobs <- resourceScanJob{index: index, url: resourceURL}:
		}
	}
}

func resourceWorker(ctx context.Context, wg *sync.WaitGroup, jobs <-chan resourceScanJob, results chan<- resourceScanResult, targetURL string, cfg config.Config) {
	defer wg.Done()
	for job := range jobs {
		discovered := ProbeResourceWithContext(ctx, job.url, targetURL, cfg, "dictionary")
		select {
		case <-ctx.Done():
			return
		case results <- resourceScanResult{index: job.index, discovery: discovered}:
		}
	}
}

func closeResourceResults(results chan<- resourceScanResult, wg *sync.WaitGroup) {
	wg.Wait()
	close(results)
}

func collectResourceResults(results <-chan resourceScanResult) []resourceScanResult {
	items := make([]resourceScanResult, 0)
	for result := range results {
		items = append(items, result)
	}
	return items
}

// ProbeResource sends one GET request and records metadata needed by resource discovery output.
func ProbeResource(rawURL string, baseURL string, cfg config.Config, discoverSource string) discoveredResource {
	return ProbeResourceWithContext(context.Background(), rawURL, baseURL, cfg, discoverSource)
}

func ProbeResourceWithContext(ctx context.Context, rawURL string, baseURL string, cfg config.Config, discoverSource string) discoveredResource {
	start := time.Now()
	record := newResourceRecord(rawURL, baseURL, discoverSource)
	req, err := newResourceRequest(ctx, rawURL, cfg)
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

func newResourceRequest(ctx context.Context, rawURL string, cfg config.Config) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
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

// buildSoft404Baseline probes one random path so later 200-like missing pages can be filtered.
func buildSoft404Baseline(ctx context.Context, baseURL *url.URL, targetURL string, cfg config.Config) soft404Baseline {
	if !cfg.EnableSoft404Detection {
		return soft404Baseline{}
	}

	baselineURL := randomSoft404URL(baseURL)
	discovered := ProbeResourceWithContext(ctx, baselineURL, targetURL, cfg, "soft404-baseline")
	if discovered.record.FetchError != "" || discovered.content == "" {
		return soft404Baseline{}
	}

	return soft404Baseline{
		enabled:       true,
		statusCode:    discovered.record.StatusCode,
		contentLength: discovered.record.ContentLength,
		title:         extractHTMLTitle(discovered.content),
		bodyHash:      hashBody(normalizeBodyForHash(discovered.content)),
	}
}

func randomSoft404URL(baseURL *url.URL) string {
	probe := *baseURL
	probe.RawQuery = ""
	probe.Fragment = ""
	probe.Path = path.Join("/", "apiextractor-soft-404-"+time.Now().UTC().Format("20060102150405.000000000"))
	return probe.String()
}

// isSoft404 uses exact body hash first, then title plus similar length as a lightweight fallback.
func isSoft404(discovered discoveredResource, baseline soft404Baseline) bool {
	if !baseline.enabled || discovered.record.FetchError != "" {
		return false
	}
	if discovered.record.StatusCode != baseline.statusCode {
		return false
	}
	if discovered.record.StatusCode == http.StatusUnauthorized || discovered.record.StatusCode == http.StatusForbidden {
		return false
	}

	title := extractHTMLTitle(discovered.content)
	bodyHash := hashBody(normalizeBodyForHash(discovered.content))
	if baseline.bodyHash != "" && bodyHash == baseline.bodyHash {
		return true
	}
	if baseline.title != "" && strings.EqualFold(title, baseline.title) && similarLength(discovered.record.ContentLength, baseline.contentLength) {
		return true
	}
	return false
}

func extractHTMLTitle(body string) string {
	lower := strings.ToLower(body)
	tagStart := strings.Index(lower, "<title")
	if tagStart < 0 {
		return ""
	}
	openEndOffset := strings.Index(lower[tagStart:], ">")
	if openEndOffset < 0 {
		return ""
	}
	titleStart := tagStart + openEndOffset + 1
	end := strings.Index(lower[titleStart:], "</title>")
	if end < 0 {
		return ""
	}
	return strings.TrimSpace(body[titleStart : titleStart+end])
}

func normalizeBodyForHash(body string) string {
	return strings.Join(strings.Fields(body), " ")
}

func hashBody(body string) string {
	sum := sha1.Sum([]byte(body))
	return hex.EncodeToString(sum[:])
}

func similarLength(left int, right int) bool {
	if left == right {
		return true
	}
	maxValue := maxInt(left, right)
	if maxValue == 0 {
		return true
	}
	diff := maxInt(left, right) - minInt(left, right)
	return float64(diff)/float64(maxValue) <= 0.05
}

func maxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}

func minInt(left int, right int) int {
	if left < right {
		return left
	}
	return right
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
