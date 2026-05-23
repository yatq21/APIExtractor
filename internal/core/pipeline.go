package core

import (
	"crypto/sha1"
	"net/http"
	"net/url"
	"strings"
	"time"

	"apiextractor/internal/config"
	"apiextractor/internal/model"
	"apiextractor/internal/urlutil"
)

// Run executes the scan pipeline and produces a structured module-three-oriented result.
func Run(targetURL string, cfg config.Config) model.ScanResult {
	result := model.ScanResult{
		Meta: model.ScanMeta{
			ToolName:      "APIExtractor",
			ToolVersion:   "0.1.2",
			SchemaVersion: "0.1.2",
			ScanID:        buildScanID(targetURL),
			ScanTime:      time.Now().Format("2006-01-02 15:04:05"),
			LogLevel:      cfg.LogLevel,
		},
		Target: model.TargetInfo{
			URL:    targetURL,
			Origin: urlutil.Origin(targetURL),
			ConfigSummary: model.ConfigSummary{
				SameOrigin:         cfg.SameOrigin,
				RequestConcurrency: cfg.RequestConcurrency,
				TimeoutSeconds:     int(cfg.Timeout.Seconds()),
				MaxResources:       cfg.MaxResources,
				MaxDepth:           cfg.MaxDepth,
				MaxSourceFiles:     cfg.MaxSourceFiles,
				MaxResponsePreview: cfg.MaxResponsePreview,
				BuiltinWordlist:    !cfg.DisableBuiltinWordlist,
				LocalWordlist:      cfg.WordlistPath,
				OutputFormat:       cfg.OutputFormat,
			},
		},
		TargetURL:    targetURL,
		TargetOrigin: urlutil.Origin(targetURL),
		Wordlists:    builtinWordlists(),
	}

	html, pageErr := FetchURL(targetURL, cfg)
	if pageErr != nil {
		result.Errors = append(result.Errors, truncateError(pageErr.Error()))
		return result
	}

	wordlistTargets, wordlistMeta, wordlistErr := LoadWordlists(targetURL, cfg)
	if wordlistErr != nil {
		result.Errors = append(result.Errors, truncateError(wordlistErr.Error()))
	} else if len(wordlistMeta) > 0 {
		result.Wordlists = wordlistMeta
	}
	scannedResources, budgetHits := ScanDirectoryResources(wordlistTargets, cfg)
	result.BudgetHits = append(result.BudgetHits, budgetHits...)

	sourceURLs := ExtractSourceURLs(html, targetURL, cfg.SameOrigin)
	sourceFiles := FetchSourceFiles(sourceURLs, cfg)
	result.JSFiles = collectSourceURLs(sourceFiles)
	result.Resources = buildResources(targetURL, html, sourceFiles)
	result.Resources = mergeResources(result.Resources, scannedResources)

	rawCandidates := ExtractAll(html, targetURL, sourceFiles, result.Resources)
	rawCandidates = append(rawCandidates, resourcesToCandidates(scannedResources)...)
	rawCandidates = append(rawCandidates, extractCandidatesFromScannedResources(scannedResources)...)
	result.Candidates = NormalizeCandidates(rawCandidates, targetURL, cfg.SameOrigin)
	result.APIResults = AnalyzeResults(RequestAll(result.Candidates, cfg), cfg)

	recoveredRaw := ExtractResponseCandidates(result.APIResults)
	recoveredCandidates := NormalizeCandidates(recoveredRaw, targetURL, cfg.SameOrigin)
	if len(recoveredCandidates) > 0 {
		result.Candidates = mergeCandidates(result.Candidates, recoveredCandidates)
		recoveredResults := AnalyzeResults(RequestAll(filterUnverifiedCandidates(result.Candidates, result.APIResults), cfg), cfg)
		result.APIResults = append(result.APIResults, recoveredResults...)
	}
	result.Summary = buildSummary(result)

	return result
}

func builtinWordlists() []model.WordlistMeta {
	return []model.WordlistMeta{
		{
			WordlistName:    "builtin-mini",
			WordlistVersion: "0.1.2",
			SourceType:      "builtin",
			UpdatedAt:       "2026-05-22",
			EntryCount:      0,
			Category:        "reserved",
			Maintainer:      "APIExtractor",
		},
	}
}

func collectSourceURLs(files []model.SourceFile) []string {
	urls := make([]string, 0, len(files))
	for _, file := range files {
		urls = append(urls, file.URL)
	}
	return urls
}

func buildSummary(result model.ScanResult) model.Summary {
	summary := model.Summary{
		CandidateCount:    len(result.Candidates),
		RecoveredCount:    countRecoveredCandidates(result.Candidates),
		JSFileCount:       len(result.JSFiles),
		ResourceCount:     len(result.Resources),
		BudgetHitCount:    len(result.BudgetHits),
		ResourceTypeStats: make(map[string]int),
		RiskTagStats:      make(map[string]int),
		ErrorTypeStats:    make(map[string]int),
	}

	for _, item := range result.Resources {
		if item.ShouldAnalyze {
			summary.AnalyzedResourceCount++
		}
		if item.Type != "" {
			summary.ResourceTypeStats[item.Type]++
		}
		if item.ErrorType != "" {
			summary.ErrorTypeStats[item.ErrorType]++
		}
	}

	for _, item := range result.APIResults {
		if item.ErrorReason != "" {
			summary.FailedRequests++
			if item.ErrorType != "" {
				summary.ErrorTypeStats[item.ErrorType]++
			}
			continue
		}
		summary.SuccessfulRequests++
		summary.VerifiedAPIs++
		if strings.Contains(strings.ToLower(item.ContentType), "json") {
			summary.JSONResponses++
		}
		if containsTag(item.RiskTags, "auth_required") {
			summary.AuthRequiredCount++
		}
		if containsTag(item.RiskTags, "forbidden") {
			summary.ForbiddenCount++
		}
		if containsTag(item.RiskTags, "large_json_response") {
			summary.LargeJSONCount++
		}
		summary.RiskTagCount += len(item.RiskTags)
		for _, tag := range item.RiskTags {
			summary.RiskTagStats[tag]++
		}
		for _, match := range item.SensitiveMatches {
			summary.SensitiveMatchCount += match.Count
		}
	}

	return summary
}

func buildResources(targetURL string, html string, files []model.SourceFile) []model.ResourceRecord {
	out := make([]model.ResourceRecord, 0, len(files)+1)
	out = append(out, model.ResourceRecord{
		ResourceID:    "res-target",
		URL:           targetURL,
		Path:          resourcePath(targetURL),
		Type:          "html",
		ContentType:   "text/html",
		Category:      "page",
		Source:        "entrypoint",
		SameOrigin:    true,
		ShouldAnalyze: true,
		BodyPreview:   htmlPreview(html),
		Tags:          []string{"entrypoint"},
	})
	for idx, item := range files {
		category := "resource"
		if item.SourceType == "javascript" || item.SourceType == "module" || item.SourceType == "sourcemap" || item.SourceType == "json" {
			category = "frontend-source"
		}
		out = append(out, model.ResourceRecord{
			ResourceID:    resourceID(idx + 1),
			URL:           item.URL,
			Path:          resourcePath(item.URL),
			Type:          item.SourceType,
			ContentType:   inferContentTypeFromSource(item.SourceType),
			Category:      category,
			Source:        "crawler",
			SameOrigin:    sameOriginMatch(item.URL, targetURL),
			ShouldAnalyze: item.Error == "",
			BodyPreview:   htmlPreview(item.Content),
			Tags:          resourceTags(item.SourceType, item.URL),
			FetchError:    item.Error,
			ErrorType:     item.ErrorType,
		})
	}
	return out
}

func mergeResources(base []model.ResourceRecord, extra []model.ResourceRecord) []model.ResourceRecord {
	seen := make(map[string]struct{}, len(base))
	out := append([]model.ResourceRecord(nil), base...)
	for _, item := range base {
		seen[item.URL] = struct{}{}
	}
	for _, item := range extra {
		if _, exists := seen[item.URL]; exists {
			continue
		}
		out = append(out, item)
		seen[item.URL] = struct{}{}
	}
	return out
}

func resourceID(index int) string {
	return "res-" + intString(index)
}

func containsTag(tags []string, expected string) bool {
	for _, item := range tags {
		if item == expected {
			return true
		}
	}
	return false
}

func mergeCandidates(base []model.APICandidate, recovered []model.APICandidate) []model.APICandidate {
	seen := make(map[string]struct{}, len(base))
	out := append([]model.APICandidate(nil), base...)
	for _, item := range base {
		seen[item.MethodGuess+" "+item.NormalizedURL] = struct{}{}
	}
	for _, item := range recovered {
		key := item.MethodGuess + " " + item.NormalizedURL
		if _, exists := seen[key]; exists {
			continue
		}
		item.Tags = append(item.Tags, "recovered-from-response")
		out = append(out, item)
		seen[key] = struct{}{}
	}
	return out
}

func filterUnverifiedCandidates(candidates []model.APICandidate, existing []model.APIResult) []model.APICandidate {
	seen := make(map[string]struct{}, len(existing))
	for _, item := range existing {
		seen[item.Method+" "+item.APIURL] = struct{}{}
	}
	out := make([]model.APICandidate, 0)
	for _, item := range candidates {
		method := item.MethodGuess
		if method == "" {
			method = http.MethodGet
		}
		key := method + " " + item.NormalizedURL
		if _, exists := seen[key]; exists {
			continue
		}
		out = append(out, item)
	}
	return out
}

func countRecoveredCandidates(items []model.APICandidate) int {
	count := 0
	for _, item := range items {
		if containsTag(item.Tags, "recovered-from-response") {
			count++
		}
	}
	return count
}

func resourcesToCandidates(resources []model.ResourceRecord) []model.ExtractedCandidate {
	out := make([]model.ExtractedCandidate, 0, len(resources))
	for _, item := range resources {
		if item.Type == "api_doc" || item.Type == "html" || item.Type == "json" || item.Type == "xml" || item.Type == "text" {
			out = append(out, model.ExtractedCandidate{
				RawValue:         item.URL,
				SourceResourceID: item.ResourceID,
				SourceURL:        item.URL,
				SourceType:       item.Type,
				DiscoverRule:     "directory-scan",
			})
		}
	}
	return out
}

func extractCandidatesFromScannedResources(resources []model.ResourceRecord) []model.ExtractedCandidate {
	out := make([]model.ExtractedCandidate, 0)
	for _, item := range resources {
		if item.BodyPreview == "" {
			continue
		}
		out = append(out, ExtractCandidatesFromResourceBody(item.BodyPreview, item)...)
	}
	return out
}

func buildScanID(targetURL string) string {
	sum := sha1.Sum([]byte(targetURL + "|" + time.Now().Format(time.RFC3339Nano)))
	return intString(int(sum[0])) + intString(int(sum[1])) + intString(int(sum[2])) + intString(int(sum[3]))
}

func resourcePath(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	if parsed.EscapedPath() == "" {
		return "/"
	}
	return parsed.EscapedPath()
}

func inferContentTypeFromSource(sourceType string) string {
	switch sourceType {
	case "javascript", "module":
		return "application/javascript"
	case "sourcemap":
		return "application/json"
	case "json":
		return "application/json"
	default:
		return ""
	}
}

func resourceTags(resourceType string, rawURL string) []string {
	tags := make([]string, 0, 3)
	lowerURL := strings.ToLower(rawURL)
	switch resourceType {
	case "javascript", "module":
		tags = append(tags, "script")
	case "sourcemap":
		tags = append(tags, "source-map")
	case "json":
		tags = append(tags, "data-file")
	}
	if strings.Contains(lowerURL, "chunk") || strings.Contains(lowerURL, "bundle") {
		tags = append(tags, "bundler-artifact")
	}
	return tags
}

func htmlPreview(text string) string {
	text = strings.TrimSpace(text)
	if len(text) <= 256 {
		return text
	}
	return text[:256]
}
