package core

import (
	"fmt"
	"sort"
	"strings"

	"apiextractor/internal/model"
)

// BuildExportRows converts API results into flattened rows for display.
func BuildExportRows(result model.ScanResult) []model.ExportRow {
	rows := make([]model.ExportRow, 0, len(result.APIResults))
	for _, item := range result.APIResults {
		errorText := item.ErrorReason
		if item.ErrorType != "" && item.ErrorReason != "" {
			errorText = item.ErrorType + ":" + item.ErrorReason
		}
		rows = append(rows, model.ExportRow{
			Method:       item.Method,
			URL:          item.APIURL,
			StatusCode:   item.StatusCode,
			DurationMS:   item.ResponseTimeMS,
			ResponseSize: item.ContentLength,
			ContentType:  item.ContentType,
			RiskTags:     strings.Join(item.RiskTags, ","),
			SourceType:   item.SourceType,
			Confidence:   item.Confidence,
			Category:     item.Category,
			RiskEvidence: summarizeRiskEvidence(item.RiskEvidence),
			CurlCommand:  item.CurlCommand,
			Error:        errorText,
		})
	}
	sort.SliceStable(rows, func(i, j int) bool {
		left := rowPriority(rows[i])
		right := rowPriority(rows[j])
		if left != right {
			return left > right
		}
		if rows[i].StatusCode != rows[j].StatusCode {
			return rows[i].StatusCode > rows[j].StatusCode
		}
		return rows[i].URL < rows[j].URL
	})
	return rows
}

func summarizeRiskEvidence(items []model.RiskEvidence) string {
	if len(items) == 0 {
		return ""
	}
	parts := make([]string, 0, min(2, len(items)))
	for i, item := range items {
		if i >= 2 {
			break
		}
		part := item.RiskTag
		if item.Reason != "" {
			part = fmt.Sprintf("%s:%s", item.RiskTag, item.Reason)
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, " | ")
}

func min(left int, right int) int {
	if left < right {
		return left
	}
	return right
}

func rowPriority(row model.ExportRow) int {
	score := 0
	if row.Error != "" {
		score += 1
	}
	if row.RiskTags != "" {
		score += 5
	}
	if strings.Contains(row.RiskTags, "sensitive_data_possible") || strings.Contains(row.RiskTags, "unauth_access_possible") {
		score += 5
	}
	if strings.Contains(row.RiskTags, "auth_required") || strings.Contains(row.RiskTags, "internal_api_exposed") || strings.Contains(row.RiskTags, "admin_api_exposed") {
		score += 3
	}
	if row.Confidence == "high" {
		score += 2
	}
	if row.Confidence == "low" {
		score -= 1
	}
	return score
}
