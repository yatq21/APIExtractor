package core

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"apiextractor/internal/config"
	"apiextractor/internal/model"
)

// RequestAPI verifies one normalized API candidate using a constrained GET request.
func RequestAPI(candidate model.APICandidate, cfg config.Config) model.APIResult {
	start := time.Now()
	redirectLocation := ""
	item := model.APIResult{
		CandidateID:      candidate.CandidateID,
		APIURL:           candidate.NormalizedURL,
		Method:           candidate.MethodGuess,
		Category:         candidate.Category,
		Confidence:       candidate.Confidence,
		SourceResourceID: candidate.SourceResourceID,
		SourceURL:        candidate.SourceURL,
		SourceType:       candidate.SourceType,
		CurlCommand:      buildCurlCommand(candidate, cfg),
	}

	client := &http.Client{
		Timeout: cfg.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if !cfg.FollowSameOriginRedirect {
				return http.ErrUseLastResponse
			}
			if len(via) == 0 {
				return nil
			}
			redirectLocation = req.URL.String()
			original, err := url.Parse(candidate.NormalizedURL)
			if err != nil {
				return http.ErrUseLastResponse
			}
			if !strings.EqualFold(req.URL.Scheme, original.Scheme) || !strings.EqualFold(req.URL.Host, original.Host) {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	method := candidate.MethodGuess
	if method == "" {
		method = http.MethodGet
	}
	req, err := http.NewRequest(method, candidate.NormalizedURL, nil)
	if err != nil {
		item.ErrorType = classifyError(err)
		item.ErrorReason = truncateError(err.Error())
		return item
	}

	for key, value := range cfg.DefaultHeaders {
		req.Header.Set(key, value)
	}
	if cfg.Cookies != "" {
		req.Header.Set("Cookie", cfg.Cookies)
	}

	resp, err := client.Do(req)
	if err != nil {
		item.ErrorType = classifyError(err)
		item.ErrorReason = truncateError(err.Error())
		item.ResponseTimeMS = time.Since(start).Milliseconds()
		return item
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, cfg.MaxResponseScanBytes))
	if err != nil {
		item.ErrorType = classifyError(err)
		item.ErrorReason = truncateError(err.Error())
		return item
	}

	item.StatusCode = resp.StatusCode
	item.ContentType = resp.Header.Get("Content-Type")
	item.ResponseTimeMS = time.Since(start).Milliseconds()
	item.ContentLength = len(body)
	item.RedirectLocation = redirectLocation
	if item.RedirectLocation == "" {
		item.RedirectLocation = resp.Header.Get("Location")
	}
	item.ResponseSample = previewBody(body, cfg.MaxResponsePreview)

	return item
}

// RequestAll verifies all normalized API candidates with bounded concurrency.
func RequestAll(candidates []model.APICandidate, cfg config.Config) []model.APIResult {
	if len(candidates) == 0 {
		return nil
	}
	concurrency := cfg.RequestConcurrency
	if concurrency <= 0 {
		concurrency = 1
	}

	results := make([]model.APIResult, len(candidates))
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for i, item := range candidates {
		wg.Add(1)
		go func(index int, candidate model.APICandidate) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			results[index] = RequestAPI(candidate, cfg)
		}(i, item)
	}
	wg.Wait()
	return results
}

func previewBody(body []byte, limit int) string {
	compact := strings.TrimSpace(string(bytes.TrimSpace(body)))
	if limit <= 0 {
		limit = 2 << 10
	}
	if len(compact) <= limit {
		return compact
	}
	return compact[:limit]
}

func buildCurlCommand(candidate model.APICandidate, cfg config.Config) string {
	method := candidate.MethodGuess
	if method == "" {
		method = http.MethodGet
	}
	parts := []string{
		"curl",
		"-X", method,
	}

	headerKeys := make([]string, 0, len(cfg.DefaultHeaders))
	for key := range cfg.DefaultHeaders {
		headerKeys = append(headerKeys, key)
	}
	sort.Strings(headerKeys)
	for _, key := range headerKeys {
		parts = append(parts, "-H", shellQuote(fmt.Sprintf("%s: %s", key, cfg.DefaultHeaders[key])))
	}
	if cfg.Cookies != "" {
		parts = append(parts, "-H", shellQuote("Cookie: "+cfg.Cookies))
	}
	parts = append(parts, shellQuote(candidate.NormalizedURL))
	return strings.Join(parts, " ")
}

func shellQuote(value string) string {
	value = strings.ReplaceAll(value, `'`, `'\''`)
	return "'" + value + "'"
}

func classifyError(err error) string {
	if err == nil {
		return ""
	}
	lower := strings.ToLower(err.Error())
	switch {
	case strings.Contains(lower, "timeout"):
		return "timeout"
	case strings.Contains(lower, "no such host"):
		return "dns_error"
	case strings.Contains(lower, "connection refused"):
		return "connection_refused"
	case strings.Contains(lower, "redirect"):
		return "redirect_error"
	case strings.Contains(lower, "unsupported protocol"):
		return "invalid_url"
	default:
		return "request_error"
	}
}

func truncateError(message string) string {
	message = strings.TrimSpace(message)
	if len(message) <= 160 {
		return message
	}
	return message[:160]
}
