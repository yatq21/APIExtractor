package exporter

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"apiextractor/internal/config"
	"apiextractor/internal/core"
	"apiextractor/internal/model"
)

// ExportResults exports a scan result according to the configured output format.
func ExportResults(result model.ScanResult, cfg config.Config) error {
	switch cfg.OutputFormat {
	case "json":
		return exportJSON(result, cfg.OutputPath)
	case "table":
		printTable(result)
		return nil
	default:
		return fmt.Errorf("unsupported output format: %s", cfg.OutputFormat)
	}
}

func printTable(result model.ScanResult) {
	rows := core.BuildExportRows(result)
	fmt.Printf("Target: %s\n", result.TargetURL)
	fmt.Printf("Origin: %s  LogLevel: %s  ScanID: %s\n", result.TargetOrigin, result.Meta.LogLevel, result.Meta.ScanID)
	fmt.Printf(
		"Candidates: %d  Recovered: %d  Verified: %d  Success: %d  Failed: %d\n",
		result.Summary.CandidateCount,
		result.Summary.RecoveredCount,
		result.Summary.VerifiedAPIs,
		result.Summary.SuccessfulRequests,
		result.Summary.FailedRequests,
	)
	fmt.Printf(
		"Resources: %d  Analyzed: %d  BudgetHits: %d  Concurrency: %d  Timeout: %ds\n",
		result.Summary.ResourceCount,
		result.Summary.AnalyzedResourceCount,
		result.Summary.BudgetHitCount,
		result.Target.ConfigSummary.RequestConcurrency,
		result.Target.ConfigSummary.TimeoutSeconds,
	)
	highlights := buildHighlights(result)
	if len(highlights) > 0 {
		fmt.Printf("Highlights: %s\n", strings.Join(highlights, " | "))
	}
	fmt.Printf("%-4s %-6s %-3s %-7s %-8s %-12s %-16s %-12s %-12s %s\n", "NO", "METHOD", "SC", "TIME", "SIZE", "TYPE", "RISK", "CONF", "CAT", "URL")
	for idx, row := range rows {
		fmt.Printf(
			"%-4d %-6s %-3d %-7d %-8d %-12s %-16s %-12s %-12s %s\n",
			idx+1,
			row.Method,
			row.StatusCode,
			row.DurationMS,
			row.ResponseSize,
			shortText(row.ContentType, 12),
			shortText(row.RiskTags, 16),
			shortText(row.Confidence, 12),
			shortText(row.Category, 12),
			row.URL,
		)
		if row.SourceType != "" {
			fmt.Printf("     source: %s\n", row.SourceType)
		}
		if row.RiskEvidence != "" {
			fmt.Printf("     evidence: %s\n", shortText(row.RiskEvidence, 160))
		}
		if row.CurlCommand != "" {
			fmt.Printf("     curl: %s\n", shortText(row.CurlCommand, 200))
		}
		if row.Error != "" {
			fmt.Printf("     error: %s\n", row.Error)
		}
	}
	if len(result.BudgetHits) > 0 {
		fmt.Printf("BudgetHits: %s\n", strings.Join(result.BudgetHits, ", "))
	}
	if len(result.Summary.ResourceTypeStats) > 0 {
		fmt.Printf("ResourceStats: %s\n", formatStats(result.Summary.ResourceTypeStats))
	}
	if len(result.Summary.RiskTagStats) > 0 {
		fmt.Printf("RiskStats: %s\n", formatStats(result.Summary.RiskTagStats))
	}
	if len(result.Summary.ErrorTypeStats) > 0 {
		fmt.Printf("ErrorStats: %s\n", formatStats(result.Summary.ErrorTypeStats))
	}
}

func exportJSON(result model.ScanResult, outputPath string) error {
	if outputPath == "" {
		outputPath = "output/result.json"
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(outputPath, data, 0o644)
}

func shortText(value string, limit int) string {
	if value == "" {
		return "-"
	}
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}

func formatStats(stats map[string]int) string {
	items := make([]string, 0, len(stats))
	for key, value := range stats {
		items = append(items, fmt.Sprintf("%s=%d", key, value))
	}
	sort.Strings(items)
	return strings.Join(items, ", ")
}

func buildHighlights(result model.ScanResult) []string {
	items := make([]string, 0, 4)
	if result.Summary.RiskTagStats["sensitive_data_possible"] > 0 {
		items = append(items, fmt.Sprintf("sensitive_hits=%d", result.Summary.RiskTagStats["sensitive_data_possible"]))
	}
	if result.Summary.RiskTagStats["unauth_access_possible"] > 0 {
		items = append(items, fmt.Sprintf("unauth_candidates=%d", result.Summary.RiskTagStats["unauth_access_possible"]))
	}
	if result.Summary.RiskTagStats["internal_api_exposed"] > 0 {
		items = append(items, fmt.Sprintf("internal_exposed=%d", result.Summary.RiskTagStats["internal_api_exposed"]))
	}
	if result.Summary.RiskTagStats["swagger_exposed"] > 0 {
		items = append(items, fmt.Sprintf("api_docs=%d", result.Summary.RiskTagStats["swagger_exposed"]))
	}
	if result.Summary.FailedRequests > 0 {
		items = append(items, fmt.Sprintf("request_failures=%d", result.Summary.FailedRequests))
	}
	return items
}
