package exporter

import (
	"encoding/json"
	"fmt"
	"os"

	"apiextractor/internal/config"
	"apiextractor/internal/core"
	"apiextractor/internal/model"
)

// ExportResults 根据配置的输出格式分发扫描结果。
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

// printTable 将请求结果渲染为紧凑的终端表格。
func printTable(result model.ScanResult) {
	rows := core.BuildExportRows(result)
	fmt.Printf("%-4s %-3s %-7s %-8s %-12s %-8s %s\n", "NO", "SC", "TIME", "SIZE", "SEVERITY", "TYPE", "URL")
	for idx, row := range rows {
		fmt.Printf(
			"%-4d %-3d %-7d %-8d %-12s %-8s %s\n",
			idx+1,
			row.StatusCode,
			row.DurationMS,
			row.ResponseSize,
			row.Severity,
			shortContentType(row.ContentType),
			row.URL,
		)
		if row.Error != "" {
			fmt.Printf("     error: %s\n", row.Error)
		}
	}
}

// exportJSON 将完整扫描结果写入缩进格式的 JSON 文件。
func exportJSON(result model.ScanResult, outputPath string) error {
	if outputPath == "" {
		outputPath = "output/result.json"
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(outputPath, data, 0o644)
}

// shortContentType 截断过长的 Content-Type，便于表格展示。
func shortContentType(contentType string) string {
	if contentType == "" {
		return "-"
	}
	if len(contentType) <= 8 {
		return contentType
	}
	return contentType[:8]
}
