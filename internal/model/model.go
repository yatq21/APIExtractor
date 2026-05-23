package model

// SourceFile 保存已下载的 JavaScript 源码及其抓取错误。
type SourceFile struct {
	URL        string `json:"url"`
	SourceType string `json:"source_type,omitempty"`
	Content    string `json:"content,omitempty"`
	Error      string `json:"error,omitempty"`
}

type ResourceRecord struct {
	URL            string `json:"url"`
	FinalURL       string `json:"final_url,omitempty"`
	Method         string `json:"method"`
	StatusCode     int    `json:"status_code"`
	ContentLength  int    `json:"content_length"`
	ContentType    string `json:"content_type,omitempty"`
	DurationMS     int64  `json:"duration_ms"`
	ResourceType   string `json:"resource_type"`
	DiscoverSource string `json:"discover_source"`
	SameOrigin     bool   `json:"same_origin"`
	ShouldAnalyze  bool   `json:"should_analyze"`
	FetchError     string `json:"fetch_error,omitempty"`
}

// RequestResult 记录单个接口请求的响应元信息和响应体预览。
type RequestResult struct {
	URL          string `json:"url"`
	Method       string `json:"method"`
	StatusCode   int    `json:"status_code"`
	ContentType  string `json:"content_type,omitempty"`
	DurationMS   int64  `json:"duration_ms"`
	ResponseSize int    `json:"response_size"`
	BodyPreview  string `json:"body_preview,omitempty"`
	Error        string `json:"error,omitempty"`
}

// AnalysisResult 保存单个接口的初步风险分析结果。
type AnalysisResult struct {
	URL      string `json:"url"`
	Severity string `json:"severity"`
	Reason   string `json:"reason"`
}

// Summary 保存一次扫描完成后的汇总计数。
type Summary struct {
	CandidateCount          int `json:"candidate_count"`
	JSFileCount             int `json:"js_file_count"`
	ResourceCount           int `json:"resource_count"`
	AnalyzableResourceCount int `json:"analyzable_resource_count"`
	SuccessfulRequests      int `json:"successful_requests"`
	FailedRequests          int `json:"failed_requests"`
	JSONResponses           int `json:"json_responses"`
}

// ScanResult 是扫描流程产出的顶层结果对象。
type ScanResult struct {
	TargetURL      string           `json:"target_url"`
	JSFiles        []string         `json:"js_files"`
	Resources      []ResourceRecord `json:"resources"`
	Candidates     []string         `json:"candidates"`
	RequestResults []RequestResult  `json:"request_results"`
	Analysis       []AnalysisResult `json:"analysis"`
	Summary        Summary          `json:"summary"`
	Errors         []string         `json:"errors,omitempty"`
}

// ExportRow 是供终端和文件导出使用的扁平化结果行。
type ExportRow struct {
	URL          string
	StatusCode   int
	DurationMS   int64
	ResponseSize int
	ContentType  string
	Severity     string
	Reason       string
	Error        string
}
