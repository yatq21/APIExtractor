package core

import (
	"strings"

	"apiextractor/internal/config"
	"apiextractor/internal/model"
)

// Run 执行完整的静态发现和直接请求扫描流程。
func Run(targetURL string, cfg config.Config) model.ScanResult {
	result := model.ScanResult{
		TargetURL: targetURL,
	}

	html, pageErr := FetchURL(targetURL, cfg)
	if pageErr != nil {
		result.Errors = append(result.Errors, pageErr.Error())
		return result
	}

	resourceRecords, resourceFiles, resourceErrors := runResourceDiscovery(targetURL, cfg)
	result.Errors = append(result.Errors, resourceErrors...)

	sourceURLs := collectPipelineSourceURLs(html, targetURL, resourceFiles, cfg)
	sourceFiles := FetchSourceFiles(sourceURLs, cfg)
	allSourceFiles := mergeSourceFiles(resourceFiles, sourceFiles)
	rawCandidates := ExtractAll(html, allSourceFiles)
	result.Candidates = NormalizeURLs(rawCandidates, targetURL, cfg.SameOrigin)
	result.RequestResults = RequestAll(result.Candidates, cfg)
	result.Analysis = AnalyzeResults(result.RequestResults)
	result.JSFiles = collectSourceURLs(sourceFiles)
	result.Resources = resourceRecords
	result.Summary = buildSummary(result)

	return result
}

func runResourceDiscovery(targetURL string, cfg config.Config) ([]model.ResourceRecord, []model.SourceFile, []string) {
	if !cfg.EnableDirectoryScan {
		return nil, nil, nil
	}

	dictionary, dictionaryErrs := LoadDictionary(cfg)
	errors := make([]string, 0, len(dictionaryErrs))
	for _, err := range dictionaryErrs {
		errors = append(errors, err.Error())
	}

	records, files := DiscoverResources(targetURL, dictionary, cfg)
	return records, files, errors
}

func collectPipelineSourceURLs(html string, targetURL string, resourceFiles []model.SourceFile, cfg config.Config) []string {
	sourceURLs := ExtractSourceURLs(html, targetURL, cfg.SameOrigin)
	resourceSourceURLs := collectResourceSourceURLs(resourceFiles, cfg.SameOrigin)
	return appendUniqueURLs(sourceURLs, resourceSourceURLs...)
}

func mergeSourceFiles(first []model.SourceFile, second []model.SourceFile) []model.SourceFile {
	files := make([]model.SourceFile, 0, len(first)+len(second))
	files = append(files, first...)
	files = append(files, second...)
	return files
}

func collectSourceURLs(files []model.SourceFile) []string {
	urls := make([]string, 0, len(files))
	for _, file := range files {
		urls = append(urls, file.URL)
	}
	return urls
}

func collectResourceSourceURLs(files []model.SourceFile, sameOrigin bool) []string {
	urls := make([]string, 0)
	for _, file := range files {
		switch file.SourceType {
		case "html":
			urls = appendUniqueURLs(urls, ExtractSourceURLs(file.Content, file.URL, sameOrigin)...)
		case "javascript", "chunk_js", "source_map", "json", "api_doc", "text":
			urls = appendUniqueURLs(urls, ExtractNestedSourceURLs(file.Content, file.URL, sameOrigin)...)
		}
	}
	return urls
}

func appendUniqueURLs(base []string, items ...string) []string {
	seen := make(map[string]struct{}, len(base)+len(items))
	for _, item := range base {
		seen[item] = struct{}{}
	}
	for _, item := range items {
		if _, exists := seen[item]; exists {
			continue
		}
		seen[item] = struct{}{}
		base = append(base, item)
	}
	return base
}

// buildSummary 根据扫描结果生成汇总计数。
func buildSummary(result model.ScanResult) model.Summary {
	summary := model.Summary{
		CandidateCount: len(result.Candidates),
		JSFileCount:    len(result.JSFiles),
		ResourceCount:  len(result.Resources),
	}

	for _, resource := range result.Resources {
		if resource.ShouldAnalyze {
			summary.AnalyzableResourceCount++
		}
	}

	for _, item := range result.RequestResults {
		if item.Error != "" {
			summary.FailedRequests++
			continue
		}
		summary.SuccessfulRequests++
		if strings.Contains(item.ContentType, "json") {
			summary.JSONResponses++
		}
	}

	return summary
}
