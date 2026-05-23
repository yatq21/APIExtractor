package model

// ScanMeta stores high-level metadata about a scan run.
type ScanMeta struct {
	ToolName      string `json:"tool_name"`
	ToolVersion   string `json:"tool_version"`
	SchemaVersion string `json:"schema_version"`
	ScanID        string `json:"scan_id,omitempty"`
	ScanTime      string `json:"scan_time,omitempty"`
	LogLevel      string `json:"log_level,omitempty"`
}

// ConfigSummary stores the effective runtime configuration for one scan.
type ConfigSummary struct {
	SameOrigin         bool   `json:"same_origin"`
	RequestConcurrency int    `json:"request_concurrency,omitempty"`
	TimeoutSeconds     int    `json:"timeout_seconds,omitempty"`
	MaxResources       int    `json:"max_resources,omitempty"`
	MaxDepth           int    `json:"max_depth,omitempty"`
	MaxSourceFiles     int    `json:"max_source_files,omitempty"`
	MaxResponsePreview int    `json:"max_response_preview,omitempty"`
	BuiltinWordlist    bool   `json:"builtin_wordlist"`
	LocalWordlist      string `json:"local_wordlist,omitempty"`
	OutputFormat       string `json:"output_format,omitempty"`
}

// TargetInfo stores target and execution summary metadata.
type TargetInfo struct {
	URL           string        `json:"url"`
	Origin        string        `json:"origin,omitempty"`
	ConfigSummary ConfigSummary `json:"config_summary"`
}

// ResourceRecord stores discovered resource metadata.
type ResourceRecord struct {
	ResourceID     string   `json:"resource_id,omitempty"`
	URL            string   `json:"url"`
	Path           string   `json:"path,omitempty"`
	Type           string   `json:"resource_type,omitempty"`
	ContentType    string   `json:"content_type,omitempty"`
	StatusCode     int      `json:"status_code,omitempty"`
	ContentLength  int      `json:"content_length,omitempty"`
	ResponseTimeMS int64    `json:"response_time,omitempty"`
	Category       string   `json:"category,omitempty"`
	Source         string   `json:"discover_source,omitempty"`
	SameOrigin     bool     `json:"same_origin,omitempty"`
	ShouldAnalyze  bool     `json:"should_analyze,omitempty"`
	BodyPreview    string   `json:"body_preview,omitempty"`
	Tags           []string `json:"tags,omitempty"`
	FetchError     string   `json:"fetch_error,omitempty"`
	ErrorType      string   `json:"error_type,omitempty"`
}

// APICandidate stores a normalized API candidate before verification.
type APICandidate struct {
	CandidateID      string   `json:"candidate_id,omitempty"`
	RawValue         string   `json:"raw_value"`
	NormalizedURL    string   `json:"normalized_url,omitempty"`
	MethodGuess      string   `json:"method_guess,omitempty"`
	Category         string   `json:"category,omitempty"`
	SourceResourceID string   `json:"source_resource_id,omitempty"`
	SourceURL        string   `json:"source_url,omitempty"`
	SourceType       string   `json:"source_type,omitempty"`
	DiscoverRule     string   `json:"discover_rule,omitempty"`
	SameOrigin       bool     `json:"same_origin"`
	Confidence       string   `json:"confidence,omitempty"`
	Tags             []string `json:"tags,omitempty"`
	IsParameterized  bool     `json:"is_parameterized,omitempty"`
	Path             string   `json:"path,omitempty"`
	SampleQuery      string   `json:"sample_query,omitempty"`
}

// ExtractedCandidate stores a raw candidate plus the source context it came from.
type ExtractedCandidate struct {
	RawValue         string   `json:"raw_value"`
	MethodHint       string   `json:"method_hint,omitempty"`
	HintTags         []string `json:"hint_tags,omitempty"`
	SourceResourceID string   `json:"source_resource_id,omitempty"`
	SourceURL        string   `json:"source_url,omitempty"`
	SourceType       string   `json:"source_type,omitempty"`
	DiscoverRule     string   `json:"discover_rule,omitempty"`
}

// SourceFile stores downloaded source text such as JavaScript or sourcemaps.
type SourceFile struct {
	URL        string `json:"url"`
	SourceType string `json:"source_type,omitempty"`
	Content    string `json:"content,omitempty"`
	Error      string `json:"error,omitempty"`
	ErrorType  string `json:"error_type,omitempty"`
}

// SensitiveMatch stores a masked sensitive data hit.
type SensitiveMatch struct {
	MatchType    string `json:"match_type"`
	MatchScope   string `json:"match_scope"`
	Count        int    `json:"count"`
	MaskedSample string `json:"masked_sample,omitempty"`
	Confidence   string `json:"confidence,omitempty"`
}

// RiskEvidence stores the minimum evidence for a risk tag.
type RiskEvidence struct {
	RiskTag        string `json:"risk_tag"`
	EvidenceType   string `json:"evidence_type"`
	EvidenceSource string `json:"evidence_source,omitempty"`
	Reason         string `json:"reason,omitempty"`
	MaskedSample   string `json:"masked_sample,omitempty"`
	Confidence     string `json:"confidence,omitempty"`
	StatusCode     int    `json:"status_code,omitempty"`
	ContentType    string `json:"content_type,omitempty"`
	ContentLength  int    `json:"content_length,omitempty"`
}

// APIResult stores the verification result for one normalized API candidate.
type APIResult struct {
	ResultID         string           `json:"result_id,omitempty"`
	CandidateID      string           `json:"candidate_id,omitempty"`
	APIURL           string           `json:"api_url"`
	Method           string           `json:"method"`
	Category         string           `json:"category,omitempty"`
	StatusCode       int              `json:"status_code"`
	ContentLength    int              `json:"content_length"`
	ResponseTimeMS   int64            `json:"response_time"`
	ContentType      string           `json:"content_type,omitempty"`
	RedirectLocation string           `json:"redirect_location,omitempty"`
	ResponseSample   string           `json:"response_sample,omitempty"`
	SourceURL        string           `json:"source_url,omitempty"`
	SourceType       string           `json:"source_type,omitempty"`
	ErrorReason      string           `json:"error_reason,omitempty"`
	ErrorType        string           `json:"error_type,omitempty"`
	RiskTags         []string         `json:"risk_tags,omitempty"`
	SensitiveMatches []SensitiveMatch `json:"sensitive_matches,omitempty"`
	SourceResourceID string           `json:"source_resource_id,omitempty"`
	Confidence       string           `json:"confidence,omitempty"`
	RiskEvidence     []RiskEvidence   `json:"risk_evidence,omitempty"`
	RiskHints        []string         `json:"risk_hints,omitempty"`
	CurlCommand      string           `json:"curl_command,omitempty"`
}

// Summary stores top-level counters for the scan report.
type Summary struct {
	CandidateCount        int            `json:"candidate_count"`
	RecoveredCount        int            `json:"recovered_count"`
	JSFileCount           int            `json:"js_file_count"`
	ResourceCount         int            `json:"resource_count"`
	AnalyzedResourceCount int            `json:"analyzed_resource_count"`
	BudgetHitCount        int            `json:"budget_hit_count"`
	SuccessfulRequests    int            `json:"successful_requests"`
	FailedRequests        int            `json:"failed_requests"`
	JSONResponses         int            `json:"json_responses"`
	VerifiedAPIs          int            `json:"verified_apis"`
	SensitiveMatchCount   int            `json:"sensitive_match_count"`
	RiskTagCount          int            `json:"risk_tag_count"`
	AuthRequiredCount     int            `json:"auth_required_count"`
	ForbiddenCount        int            `json:"forbidden_count"`
	LargeJSONCount        int            `json:"large_json_count"`
	ResourceTypeStats     map[string]int `json:"resource_type_stats,omitempty"`
	RiskTagStats          map[string]int `json:"risk_tag_stats,omitempty"`
	ErrorTypeStats        map[string]int `json:"error_type_stats,omitempty"`
}

// ScanResult is the top-level report object.
type ScanResult struct {
	Meta         ScanMeta         `json:"meta"`
	Target       TargetInfo       `json:"target"`
	TargetURL    string           `json:"target_url"`
	TargetOrigin string           `json:"target_origin,omitempty"`
	Wordlists    []WordlistMeta   `json:"wordlists,omitempty"`
	JSFiles      []string         `json:"js_files"`
	Resources    []ResourceRecord `json:"resources,omitempty"`
	Candidates   []APICandidate   `json:"api_candidates"`
	APIResults   []APIResult      `json:"api_results"`
	Summary      Summary          `json:"summary"`
	Errors       []string         `json:"errors,omitempty"`
	BudgetHits   []string         `json:"budget_hits,omitempty"`
}

// WordlistMeta stores reserved manifest metadata for built-in or user-supplied wordlists.
type WordlistMeta struct {
	WordlistName    string `json:"wordlist_name,omitempty"`
	WordlistVersion string `json:"wordlist_version,omitempty"`
	SourceType      string `json:"source_type,omitempty"`
	SourceURL       string `json:"source_url,omitempty"`
	UpdatedAt       string `json:"updated_at,omitempty"`
	EntryCount      int    `json:"entry_count,omitempty"`
	SHA256          string `json:"sha256,omitempty"`
	Category        string `json:"category,omitempty"`
	Maintainer      string `json:"maintainer,omitempty"`
}

// ExportRow is a flattened row for terminal or file output.
type ExportRow struct {
	Method       string
	URL          string
	StatusCode   int
	DurationMS   int64
	ResponseSize int
	ContentType  string
	RiskTags     string
	SourceType   string
	Confidence   string
	Category     string
	RiskEvidence string
	CurlCommand  string
	Error        string
}
