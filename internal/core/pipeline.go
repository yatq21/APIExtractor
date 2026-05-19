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

	sourceURLs := ExtractSourceURLs(html, targetURL, cfg.SameOrigin)
	sourceFiles := FetchSourceFiles(sourceURLs, cfg)
	rawCandidates := ExtractAll(html, sourceFiles)
	result.Candidates = NormalizeURLs(rawCandidates, targetURL, cfg.SameOrigin)
	result.RequestResults = RequestAll(result.Candidates, cfg)
	result.Analysis = AnalyzeResults(result.RequestResults)
	result.JSFiles = collectSourceURLs(sourceFiles)
	result.Summary = buildSummary(result)

	return result
}

func collectSourceURLs(files []model.SourceFile) []string {
	urls := make([]string, 0, len(files))
	for _, file := range files {
		urls = append(urls, file.URL)
	}
	return urls
}

// buildSummary 根据扫描结果生成汇总计数。
func buildSummary(result model.ScanResult) model.Summary {
	summary := model.Summary{
		CandidateCount: len(result.Candidates),
		JSFileCount:    len(result.JSFiles),
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
