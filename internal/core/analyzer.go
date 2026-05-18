package core

import (
	"strings"

	"apiextractor/internal/model"
)

// AnalyzeResult 为单个请求结果生成初步风险标签。
func AnalyzeResult(result model.RequestResult) model.AnalysisResult {
	analysis := model.AnalysisResult{
		URL:      result.URL,
		Severity: "info",
		Reason:   "no obvious issue detected",
	}

	if result.Error != "" {
		analysis.Severity = "low"
		analysis.Reason = "request failed"
		return analysis
	}

	if result.StatusCode >= 200 && result.StatusCode < 300 && strings.Contains(strings.ToLower(result.ContentType), "json") {
		analysis.Severity = "medium"
		analysis.Reason = "JSON endpoint exposed to direct request"
		return analysis
	}

	if result.StatusCode == 401 || result.StatusCode == 403 {
		analysis.Reason = "authentication or authorization required"
		return analysis
	}

	if result.StatusCode >= 500 {
		analysis.Severity = "medium"
		analysis.Reason = "server error returned"
		return analysis
	}

	return analysis
}

// AnalyzeResults 批量分析所有请求结果。
func AnalyzeResults(results []model.RequestResult) []model.AnalysisResult {
	items := make([]model.AnalysisResult, 0, len(results))
	for _, item := range results {
		items = append(items, AnalyzeResult(item))
	}
	return items
}
