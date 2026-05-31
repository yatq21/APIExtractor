package core

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"apiextractor/internal/config"
	"apiextractor/internal/model"
)

type directoryScanJob struct {
	index int
	url   string
}

type directoryScanResult struct {
	index    int
	resource model.ResourceRecord
}

type soft404Baseline struct {
	Enabled       bool
	StatusCode    int
	ContentLength int
	Title         string
	BodyHash      string
}

// ScanDirectoryResources performs bounded concurrent checks for wordlist-driven resource discovery.
func ScanDirectoryResources(targets []string, cfg config.Config) ([]model.ResourceRecord, []string) {
	if cfg.DisableDirectoryScan || len(targets) == 0 {
		return nil, nil
	}

	limitedTargets, budgetHits := limitDirectoryTargets(targets, cfg.MaxResources)
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout*time.Duration(max(1, len(limitedTargets))))
	defer cancel()

	baseline := buildSoft404Baseline(ctx, limitedTargets, cfg)
	results := scanDirectoryTargets(ctx, limitedTargets, cfg)
	out := make([]model.ResourceRecord, 0, len(results))
	for _, result := range results {
		item := result.resource
		if item.URL == "" {
			continue
		}
		if isSoft404Resource(item, baseline) || item.FetchError != "" || item.Type == "" {
			continue
		}
		out = append(out, item)
	}
	return out, budgetHits
}

func limitDirectoryTargets(targets []string, maxResources int) ([]string, []string) {
	if maxResources <= 0 {
		maxResources = 200
	}
	if len(targets) <= maxResources {
		return targets, nil
	}
	return targets[:maxResources], []string{"max_resources_reached"}
}

// scanDirectoryTargets runs a worker pool and keeps output order stable by original target index.
func scanDirectoryTargets(ctx context.Context, targets []string, cfg config.Config) []directoryScanResult {
	workerCount := scanWorkerCount(cfg.RequestConcurrency, len(targets))
	jobs := make(chan directoryScanJob)
	results := make(chan directoryScanResult)
	var wg sync.WaitGroup

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go directoryScanWorker(ctx, &wg, jobs, results, cfg)
	}
	go produceDirectoryScanJobs(ctx, targets, jobs)
	go closeDirectoryScanResults(results, &wg)

	out := collectDirectoryScanResults(results)
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].index < out[j].index
	})
	return out
}

func scanWorkerCount(configured int, jobCount int) int {
	if jobCount <= 0 {
		return 0
	}
	if configured <= 0 {
		configured = 10
	}
	if configured > jobCount {
		return jobCount
	}
	return configured
}

func produceDirectoryScanJobs(ctx context.Context, targets []string, jobs chan<- directoryScanJob) {
	defer close(jobs)
	for index, target := range targets {
		select {
		case <-ctx.Done():
			return
		case jobs <- directoryScanJob{index: index, url: target}:
		}
	}
}

func directoryScanWorker(ctx context.Context, wg *sync.WaitGroup, jobs <-chan directoryScanJob, results chan<- directoryScanResult, cfg config.Config) {
	defer wg.Done()
	for job := range jobs {
		resource := probeResource(ctx, job.index+1, job.url, cfg)
		select {
		case <-ctx.Done():
			return
		case results <- directoryScanResult{index: job.index, resource: resource}:
		}
	}
}

func closeDirectoryScanResults(results chan<- directoryScanResult, wg *sync.WaitGroup) {
	wg.Wait()
	close(results)
}

func collectDirectoryScanResults(results <-chan directoryScanResult) []directoryScanResult {
	out := make([]directoryScanResult, 0)
	for item := range results {
		out = append(out, item)
	}
	return out
}

func probeResource(ctx context.Context, index int, rawURL string, cfg config.Config) model.ResourceRecord {
	record := model.ResourceRecord{
		ResourceID:    resourceID(index),
		URL:           rawURL,
		Path:          scannedResourcePath(rawURL),
		Category:      "directory-scan",
		Source:        "wordlist-scan",
		SameOrigin:    true,
		ShouldAnalyze: true,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
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
		record.StatusCode = resp.StatusCode
		record.FetchError = http.StatusText(resp.StatusCode)
		record.ErrorType = "http_" + strings.ToLower(strings.ReplaceAll(http.StatusText(resp.StatusCode), " ", "_"))
		return record
	}
	record.StatusCode = resp.StatusCode
	record.Type = classifyResourceType(rawURL, resp.Header.Get("Content-Type"))
	record.ContentType = resp.Header.Get("Content-Type")
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, int64(cfg.MaxResponsePreview)))
	if readErr == nil {
		record.BodyPreview = strings.TrimSpace(string(body))
		record.ContentLength = len(body)
	}
	record.Tags = discoveryTags(record.Type, rawURL, record.BodyPreview)
	return record
}

// buildSoft404Baseline probes one likely-missing path and captures a lightweight fingerprint.
func buildSoft404Baseline(ctx context.Context, targets []string, cfg config.Config) soft404Baseline {
	if !cfg.EnableSoft404Detection || len(targets) == 0 {
		return soft404Baseline{}
	}
	probeURL := soft404ProbeURL(targets[0])
	if probeURL == "" {
		return soft404Baseline{}
	}
	record := probeResource(ctx, 0, probeURL, cfg)
	if record.FetchError != "" || record.BodyPreview == "" {
		return soft404Baseline{}
	}
	return soft404Baseline{
		Enabled:       true,
		StatusCode:    record.StatusCode,
		ContentLength: record.ContentLength,
		Title:         extractHTMLTitle(record.BodyPreview),
		BodyHash:      hashBody(normalizeBodyForHash(record.BodyPreview)),
	}
}

func soft404ProbeURL(sampleTarget string) string {
	parsed, err := url.Parse(sampleTarget)
	if err != nil {
		return ""
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	parsed.Path = "/apiextractor-soft-404-" + time.Now().UTC().Format("20060102150405.000000000")
	return parsed.String()
}

// isSoft404Resource filters pseudo-success responses that match the missing-page baseline.
func isSoft404Resource(record model.ResourceRecord, baseline soft404Baseline) bool {
	if !baseline.Enabled || record.FetchError != "" {
		return false
	}
	if record.StatusCode != baseline.StatusCode {
		return false
	}
	if record.StatusCode == http.StatusUnauthorized || record.StatusCode == http.StatusForbidden {
		return false
	}
	bodyHash := hashBody(normalizeBodyForHash(record.BodyPreview))
	if baseline.BodyHash != "" && bodyHash == baseline.BodyHash {
		return true
	}
	title := extractHTMLTitle(record.BodyPreview)
	return baseline.Title != "" && strings.EqualFold(title, baseline.Title) && similarLength(record.ContentLength, baseline.ContentLength)
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

func minIntValue(left int, right int) int {
	if left < right {
		return left
	}
	return right
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
	maxValue := max(left, right)
	if maxValue == 0 {
		return true
	}
	diff := max(left, right) - minIntValue(left, right)
	return float64(diff)/float64(maxValue) <= 0.05
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

func discoveryTags(resourceType string, rawURL string, content string) []string {
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
	if resourceType == "sourcemap" && sourceMapHasSourcesOnly(content) {
		tags = append(tags, "sourcemap-sources-only")
	}
	return mergeStringTags(tags, detectFrameworkTags(rawURL, content))
}
