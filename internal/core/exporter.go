package core

import "apiextractor/internal/model"

// BuildExportRows 将扫描结果转换为适合终端或文件输出的扁平行结构。
func BuildExportRows(result model.ScanResult) []model.ExportRow {
	rows := make([]model.ExportRow, 0, len(result.RequestResults))
	for idx, item := range result.RequestResults {
		row := model.ExportRow{
			URL:          item.URL,
			StatusCode:   item.StatusCode,
			DurationMS:   item.DurationMS,
			ResponseSize: item.ResponseSize,
			ContentType:  item.ContentType,
			Error:        item.Error,
		}
		if idx < len(result.Analysis) {
			row.Severity = result.Analysis[idx].Severity
			row.Reason = result.Analysis[idx].Reason
		}
		rows = append(rows, row)
	}
	return rows
}
