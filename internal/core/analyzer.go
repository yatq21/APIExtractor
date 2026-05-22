package core

import (
	"encoding/json"
	"net/http"
	"path"
	"regexp"
	"sort"
	"strings"

	"apiextractor/internal/config"
	"apiextractor/internal/model"
)

var sensitiveRules = []struct {
	name       string
	scope      string
	pattern    *regexp.Regexp
	confidence string
}{
	{name: "email", scope: "field_value", pattern: regexp.MustCompile(`(?i)\b[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}\b`), confidence: "high"},
	{name: "phone", scope: "field_value", pattern: regexp.MustCompile(`(?i)\b1[3-9]\d{9}\b`), confidence: "medium"},
	{name: "jwt", scope: "field_value", pattern: regexp.MustCompile(`eyJ[a-zA-Z0-9_\-]+=*\.[a-zA-Z0-9_\-]+=*\.?[a-zA-Z0-9_\-+/=]*`), confidence: "high"},
	{name: "token", scope: "field_name", pattern: regexp.MustCompile(`(?i)(access_token|refresh_token|token|session|cookie)`), confidence: "medium"},
	{name: "password", scope: "field_name", pattern: regexp.MustCompile(`(?i)(password|passwd|pwd|secret)`), confidence: "high"},
	{name: "internal_ip", scope: "field_value", pattern: regexp.MustCompile(`\b(?:10\.\d{1,3}\.\d{1,3}\.\d{1,3}|172\.(?:1[6-9]|2\d|3[0-1])\.\d{1,3}\.\d{1,3}|192\.168\.\d{1,3}\.\d{1,3})\b`), confidence: "medium"},
	{name: "debug", scope: "field_value", pattern: regexp.MustCompile(`(?i)(exception|stacktrace|traceback|debug|internal server error)`), confidence: "medium"},
}

// EnrichAPIResult adds sensitive findings and risk tags to a verified API result.
func EnrichAPIResult(result model.APIResult, cfg config.Config) model.APIResult {
	matches := detectSensitiveMatches(result.ResponseSample, result.ContentType)
	result.SensitiveMatches = matches
	result.RiskTags, result.RiskEvidence, result.RiskHints = buildRiskSignals(result, matches, cfg)
	return result
}

// AnalyzeResults enriches all API verification results.
func AnalyzeResults(results []model.APIResult, cfg config.Config) []model.APIResult {
	items := make([]model.APIResult, 0, len(results))
	for _, item := range results {
		items = append(items, EnrichAPIResult(item, cfg))
	}
	return items
}

func detectSensitiveMatches(sample string, contentType string) []model.SensitiveMatch {
	if sample == "" {
		return nil
	}

	type aggregate struct {
		match model.SensitiveMatch
	}

	aggregated := make(map[string]*aggregate)
	addMatch := func(ruleName string, scope string, count int, sampleValue string, confidence string) {
		if count <= 0 {
			return
		}
		key := ruleName + ":" + scope
		existing, ok := aggregated[key]
		if !ok {
			aggregated[key] = &aggregate{
				match: model.SensitiveMatch{
					MatchType:    ruleName,
					MatchScope:   scope,
					Count:        count,
					MaskedSample: maskSample(sampleValue),
					Confidence:   confidence,
				},
			}
			return
		}
		existing.match.Count += count
	}

	if looksLikeJSON(contentType, sample) {
		for _, finding := range detectJSONSensitiveMatches(sample) {
			addMatch(finding.MatchType, finding.MatchScope, finding.Count, finding.MaskedSample, finding.Confidence)
		}
	}

	for _, rule := range sensitiveRules {
		matches := rule.pattern.FindAllString(sample, -1)
		if len(matches) == 0 {
			continue
		}
		addMatch(rule.name, rule.scope, len(matches), matches[0], rule.confidence)
	}

	out := make([]model.SensitiveMatch, 0, len(aggregated))
	for _, item := range aggregated {
		out = append(out, item.match)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].MatchType == out[j].MatchType {
			return out[i].MatchScope < out[j].MatchScope
		}
		return out[i].MatchType < out[j].MatchType
	})
	return out
}

func detectJSONSensitiveMatches(sample string) []model.SensitiveMatch {
	var payload any
	if err := json.Unmarshal([]byte(sample), &payload); err != nil {
		return nil
	}

	out := make([]model.SensitiveMatch, 0)
	appendMatches := func(ruleName string, scope string, count int, sampleValue string, confidence string) {
		if count == 0 {
			return
		}
		out = append(out, model.SensitiveMatch{
			MatchType:    ruleName,
			MatchScope:   scope,
			Count:        count,
			MaskedSample: sampleValue,
			Confidence:   confidence,
		})
	}

	keyHits := make(map[string]int)
	keySamples := make(map[string]string)
	valueHits := make(map[string]int)
	valueSamples := make(map[string]string)

	var walk func(node any)
	walk = func(node any) {
		switch value := node.(type) {
		case map[string]any:
			for key, child := range value {
				for _, rule := range sensitiveRules {
					if rule.scope != "field_name" {
						continue
					}
					if rule.pattern.MatchString(key) {
						keyHits[rule.name]++
						if keySamples[rule.name] == "" {
							keySamples[rule.name] = key
						}
					}
				}
				walk(child)
			}
		case []any:
			for _, child := range value {
				walk(child)
			}
		case string:
			for _, rule := range sensitiveRules {
				if rule.scope != "field_value" {
					continue
				}
				hits := rule.pattern.FindAllString(value, -1)
				if len(hits) == 0 {
					continue
				}
				valueHits[rule.name] += len(hits)
				if valueSamples[rule.name] == "" {
					valueSamples[rule.name] = hits[0]
				}
			}
		}
	}

	walk(payload)

	for _, rule := range sensitiveRules {
		if rule.scope == "field_name" {
			appendMatches(rule.name, rule.scope, keyHits[rule.name], keySamples[rule.name], rule.confidence)
			continue
		}
		appendMatches(rule.name, rule.scope, valueHits[rule.name], valueSamples[rule.name], rule.confidence)
	}

	return out
}

func buildRiskSignals(result model.APIResult, matches []model.SensitiveMatch, cfg config.Config) ([]string, []model.RiskEvidence, []string) {
	riskSet := make(map[string]struct{})
	evidence := make([]model.RiskEvidence, 0)
	riskHints := make([]string, 0)
	add := func(tag string, evidenceType string, source string, reason string, masked string, confidence string) {
		if _, ok := riskSet[tag]; ok {
			return
		}
		riskSet[tag] = struct{}{}
		evidence = append(evidence, model.RiskEvidence{
			RiskTag:        tag,
			EvidenceType:   evidenceType,
			EvidenceSource: source,
			Reason:         reason,
			MaskedSample:   masked,
			Confidence:     confidence,
			StatusCode:     result.StatusCode,
			ContentType:    result.ContentType,
			ContentLength:  result.ContentLength,
		})
	}
	addHint := func(hint string) {
		for _, item := range riskHints {
			if item == hint {
				return
			}
		}
		riskHints = append(riskHints, hint)
	}

	lowerURL := strings.ToLower(result.APIURL)
	lowerContentType := strings.ToLower(result.ContentType)
	lowerSample := strings.ToLower(result.ResponseSample)
	ext := strings.ToLower(path.Ext(lowerURL))

	if result.StatusCode == http.StatusUnauthorized || strings.Contains(lowerSample, "unauthorized") || strings.Contains(lowerSample, "authentication required") {
		add("auth_required", "status_or_body", result.APIURL, "status code or body indicates authentication is required", "", "high")
	}
	if result.StatusCode == http.StatusForbidden {
		add("forbidden", "status_code", result.APIURL, "response status is 403 forbidden", "", "high")
	}
	redirectLower := strings.ToLower(result.RedirectLocation)
	if strings.Contains(redirectLower, "login") || strings.Contains(redirectLower, "auth") || strings.Contains(redirectLower, "sso") {
		add("redirect_to_login", "redirect_location", result.RedirectLocation, "redirect target looks like a login or auth page", "", "medium")
	}
	if strings.Contains(lowerURL, "admin") || strings.Contains(lowerURL, "manage") || strings.Contains(lowerURL, "system") {
		if result.StatusCode != http.StatusNotFound && result.ErrorReason == "" {
			add("admin_api_exposed", "path_semantic", result.APIURL, "path semantics suggest admin or management functionality", "", "medium")
		}
	}
	if result.Category == "internal-api" && result.StatusCode >= 200 && result.StatusCode < 300 {
		add("internal_api_exposed", "category_and_status", result.APIURL, "internal semantic endpoint responded successfully", "", "medium")
	}
	if result.Category == "graphql" && result.StatusCode >= 200 && result.StatusCode < 500 {
		add("graphql_endpoint_exposed", "category_and_status", result.APIURL, "graphql endpoint appears reachable", "", "medium")
	}
	if result.Category == "api-doc" && result.StatusCode >= 200 && result.StatusCode < 400 {
		add("swagger_exposed", "category_and_status", result.APIURL, "API documentation endpoint appears reachable", "", "medium")
	}
	if result.Category == "authenticated-api" && result.StatusCode >= 200 && result.StatusCode < 300 {
		add("authenticated_api_exposed", "category_and_status", result.APIURL, "authenticated semantic endpoint responded successfully", "", "medium")
	}
	if result.StatusCode >= 200 && result.StatusCode < 300 && (strings.Contains(lowerContentType, "json") || strings.Contains(lowerContentType, "text")) && len(matches) > 0 {
		add("sensitive_data_possible", "sensitive_match", result.APIURL, "successful response contains sensitive indicators", matches[0].MaskedSample, "high")
		add("unauth_access_possible", "successful_response", result.APIURL, "request succeeded and returned meaningful response content", matches[0].MaskedSample, "medium")
	}
	if strings.Contains(lowerContentType, "json") && result.ContentLength >= cfg.LargeJSONThreshold {
		add("large_json_response", "content_length", result.APIURL, "JSON response size exceeds configured large-response threshold", "", "medium")
	}
	if strings.Contains(lowerSample, "exception") || strings.Contains(lowerSample, "stacktrace") || strings.Contains(lowerSample, "traceback") || strings.Contains(lowerSample, "debug") {
		add("debug_info_exposed", "response_sample", result.APIURL, "response sample contains debug or exception keywords", maskSample(result.ResponseSample), "medium")
	}
	if strings.Contains(lowerURL, "swagger") || strings.Contains(lowerURL, "openapi") || strings.Contains(lowerSample, "openapi") || strings.Contains(lowerSample, "swagger-ui") || strings.Contains(lowerSample, "knife4j") {
		add("swagger_exposed", "path_or_body", result.APIURL, "path or response indicates exposed API documentation", "", "medium")
	}
	if ext == ".map" || strings.Contains(lowerSample, "\"sourcescontent\"") {
		add("source_map_exposed", "path_or_body", result.APIURL, "URL or response indicates an exposed source map", "", "medium")
	}
	if ext == ".js" || ext == ".css" || strings.HasPrefix(lowerContentType, "text/css") || strings.Contains(lowerContentType, "javascript") || strings.HasPrefix(lowerContentType, "image/") {
		add("static_resource", "path_or_content_type", result.APIURL, "content appears to be a static resource rather than an API response", "", "low")
	}
	if result.Confidence == "low" {
		add("low_confidence", "normalization", result.APIURL, "candidate required heuristic normalization and should be manually reviewed", "", "low")
	}
	if strings.Contains(lowerSample, "not found") {
		addHint("body_contains_not_found")
	}
	if strings.Contains(lowerSample, "404") {
		addHint("title_contains_404")
	}
	if strings.Contains(strings.ToLower(result.RedirectLocation), "error") || strings.Contains(strings.ToLower(result.RedirectLocation), "404") {
		addHint("redirect_to_error_page")
	}

	riskTags := make([]string, 0, len(riskSet))
	for tag := range riskSet {
		riskTags = append(riskTags, tag)
	}
	sort.Strings(riskTags)
	sort.Strings(riskHints)
	return riskTags, evidence, riskHints
}

func maskSample(sample string) string {
	sample = strings.TrimSpace(sample)
	if sample == "" {
		return ""
	}
	runes := []rune(sample)
	if len(runes) <= 6 {
		return string(runes[0]) + "***"
	}
	return string(runes[:3]) + "***" + string(runes[len(runes)-3:])
}

func looksLikeJSON(contentType string, sample string) bool {
	lowerType := strings.ToLower(contentType)
	if strings.Contains(lowerType, "json") {
		return true
	}
	trimmed := strings.TrimSpace(sample)
	return strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[")
}
