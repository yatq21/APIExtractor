package core

import (
	"strings"
	"testing"

	"apiextractor/internal/model"
)

func TestBuildExportRowsIncludesRiskEvidenceSummary(t *testing.T) {
	result := model.ScanResult{
		APIResults: []model.APIResult{
			{
				Method:      "GET",
				APIURL:      "https://example.com/api/private",
				ContentType: "application/json",
				RiskTags:    []string{"auth_required"},
				CurlCommand: "curl -X GET 'https://example.com/api/private'",
				RiskEvidence: []model.RiskEvidence{
					{
						RiskTag: "auth_required",
						Reason:  "status code or body indicates authentication is required",
					},
				},
			},
		},
	}

	rows := BuildExportRows(result)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if !strings.Contains(rows[0].RiskEvidence, "auth_required") {
		t.Fatal("expected risk evidence summary in export row")
	}
	if !strings.Contains(rows[0].CurlCommand, "curl -X GET") {
		t.Fatal("expected curl command in export row")
	}
}

func TestBuildExportRowsSortsHighRiskFirst(t *testing.T) {
	result := model.ScanResult{
		APIResults: []model.APIResult{
			{
				Method:     "GET",
				APIURL:     "https://example.com/api/low",
				Confidence: "low",
			},
			{
				Method:     "GET",
				APIURL:     "https://example.com/api/high",
				Confidence: "high",
				RiskTags:   []string{"sensitive_data_possible", "unauth_access_possible"},
			},
		},
	}

	rows := BuildExportRows(result)
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0].URL != "https://example.com/api/high" {
		t.Fatal("expected high risk row to sort first")
	}
}
