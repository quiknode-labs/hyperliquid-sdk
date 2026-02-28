package hyperliquid

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// HTTPClient wraps http.Client with SDK-specific functionality.
type HTTPClient struct {
	client   *http.Client
	timeout  time.Duration
}

// NewHTTPClient creates a new HTTP client with connection pooling.
func NewHTTPClient(timeout time.Duration) *HTTPClient {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:   true,
	}

	return &HTTPClient{
		client: &http.Client{
			Transport: transport,
			Timeout:   timeout,
		},
		timeout: timeout,
	}
}

// Post sends a POST request with JSON body and returns the parsed response.
func (c *HTTPClient) Post(ctx context.Context, url string, body any) (map[string]any, error) {
	jsonData, err := json.Marshal(body)
	if err != nil {
		return nil, NewError(ErrorCodeInvalidJSON, fmt.Sprintf("failed to marshal request: %v", err))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, NewError(ErrorCodeConnectionError, fmt.Sprintf("failed to create request: %v", err))
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, NewError(ErrorCodeTimeout, fmt.Sprintf("request timed out after %v", c.timeout))
		}
		return nil, NewError(ErrorCodeConnectionError, fmt.Sprintf("request failed: %v", err))
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, NewError(ErrorCodeParseError, fmt.Sprintf("failed to read response: %v", err))
	}

	var result map[string]any
	if err := json.Unmarshal(respBody, &result); err != nil {
		// Try to parse as array for some endpoints
		var arrayResult []any
		if err2 := json.Unmarshal(respBody, &arrayResult); err2 == nil {
			return map[string]any{"data": arrayResult}, nil
		}
		return nil, NewError(ErrorCodeParseError, fmt.Sprintf("invalid JSON response: %v", err))
	}

	// Check for error in response
	if errVal, ok := result["error"]; ok && errVal != nil {
		return nil, ParseAPIError(result, resp.StatusCode)
	}

	return result, nil
}

// PostRaw sends a POST request and returns the raw response as any type.
func (c *HTTPClient) PostRaw(ctx context.Context, url string, body any) (any, error) {
	jsonData, err := json.Marshal(body)
	if err != nil {
		return nil, NewError(ErrorCodeInvalidJSON, fmt.Sprintf("failed to marshal request: %v", err))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, NewError(ErrorCodeConnectionError, fmt.Sprintf("failed to create request: %v", err))
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, NewError(ErrorCodeTimeout, fmt.Sprintf("request timed out after %v", c.timeout))
		}
		return nil, NewError(ErrorCodeConnectionError, fmt.Sprintf("request failed: %v", err))
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, NewError(ErrorCodeParseError, fmt.Sprintf("failed to read response: %v", err))
	}

	var result any
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, NewError(ErrorCodeParseError, fmt.Sprintf("invalid JSON response: %v", err))
	}

	// Check for error in response
	if m, ok := result.(map[string]any); ok {
		if errVal, ok := m["error"]; ok && errVal != nil {
			return nil, ParseAPIError(m, resp.StatusCode)
		}
	}

	return result, nil
}

// Get sends a GET request and returns the parsed response.
func (c *HTTPClient) Get(ctx context.Context, baseURL string, params map[string]string) (map[string]any, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, NewError(ErrorCodeConnectionError, fmt.Sprintf("invalid URL: %v", err))
	}

	if params != nil {
		q := u.Query()
		for k, v := range params {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, NewError(ErrorCodeConnectionError, fmt.Sprintf("failed to create request: %v", err))
	}

	resp, err := c.client.Do(req)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, NewError(ErrorCodeTimeout, fmt.Sprintf("request timed out after %v", c.timeout))
		}
		return nil, NewError(ErrorCodeConnectionError, fmt.Sprintf("request failed: %v", err))
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, NewError(ErrorCodeParseError, fmt.Sprintf("failed to read response: %v", err))
	}

	var result map[string]any
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, NewError(ErrorCodeParseError, fmt.Sprintf("invalid JSON response: %v", err))
	}

	// Check for error in response
	if errVal, ok := result["error"]; ok && errVal != nil {
		return nil, ParseAPIError(result, resp.StatusCode)
	}

	return result, nil
}

// Close closes the HTTP client.
func (c *HTTPClient) Close() {
	c.client.CloseIdleConnections()
}

// buildBaseURL extracts the token from a QuickNode URL and returns the base URL.
// Handles various URL formats:
//   - https://x.quiknode.pro/TOKEN → https://x.quiknode.pro/TOKEN
//   - https://x.quiknode.pro/TOKEN/info → https://x.quiknode.pro/TOKEN
//   - https://x.quiknode.pro/TOKEN/evm → https://x.quiknode.pro/TOKEN
func buildBaseURL(endpoint string) string {
	u, err := url.Parse(endpoint)
	if err != nil {
		return endpoint
	}

	base := fmt.Sprintf("%s://%s", u.Scheme, u.Host)
	pathParts := strings.Split(strings.Trim(u.Path, "/"), "/")

	// Known API path suffixes (not the token)
	knownPaths := map[string]bool{
		"info": true, "hypercore": true, "evm": true,
		"nanoreth": true, "ws": true, "send": true,
	}

	// Find the token (first path part that's not a known API path)
	for _, part := range pathParts {
		if part != "" && !knownPaths[part] {
			return base + "/" + part
		}
	}

	return base
}

// buildInfoURL builds the /info endpoint URL from a QuickNode endpoint.
func buildInfoURL(endpoint string) string {
	u, err := url.Parse(endpoint)
	if err != nil {
		return endpoint + "/info"
	}

	base := fmt.Sprintf("%s://%s", u.Scheme, u.Host)
	pathParts := strings.Split(strings.Trim(u.Path, "/"), "/")

	// Check if URL already ends with /info
	if len(pathParts) > 0 && pathParts[len(pathParts)-1] == "info" {
		return strings.TrimSuffix(endpoint, "/")
	}

	// Find the token
	knownPaths := map[string]bool{
		"info": true, "hypercore": true, "evm": true,
		"nanoreth": true, "ws": true, "send": true,
	}

	for _, part := range pathParts {
		if part != "" && !knownPaths[part] {
			return base + "/" + part + "/info"
		}
	}

	return base + "/info"
}

// buildHyperCoreURL builds the /hypercore endpoint URL.
func buildHyperCoreURL(endpoint string) string {
	base := buildBaseURL(endpoint)
	return base + "/hypercore"
}

// buildEVMURL builds the /evm endpoint URL.
func buildEVMURL(endpoint string) string {
	base := buildBaseURL(endpoint)
	return base + "/evm"
}

// buildSendURL builds the /send endpoint URL.
func buildSendURL(endpoint string) string {
	base := buildBaseURL(endpoint)
	return base + "/send"
}

// buildWebSocketURL builds the WebSocket URL from an endpoint.
// Handles:
//   - QuickNode: https://x.quiknode.pro/TOKEN → wss://x.quiknode.pro/TOKEN/hypercore/ws
//   - Public API: wss://api.hyperliquid.xyz/ws → wss://api.hyperliquid.xyz/ws
func buildWebSocketURL(endpoint string) (string, bool) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", false
	}

	// If already a ws/wss URL, use it
	if u.Scheme == "ws" || u.Scheme == "wss" {
		if strings.HasSuffix(u.Path, "/ws") {
			return endpoint, false
		}
		return strings.TrimSuffix(endpoint, "/") + "/ws", false
	}

	// Convert https to wss
	scheme := "wss"
	if u.Scheme == "http" {
		scheme = "ws"
	}

	base := fmt.Sprintf("%s://%s", scheme, u.Host)

	// Check if this is the public Hyperliquid API
	if strings.Contains(u.Host, "hyperliquid.xyz") || strings.Contains(u.Host, "api.hyperliquid") {
		return base + "/ws", false
	}

	// QuickNode endpoint - extract token and build /hypercore/ws path
	pathParts := strings.Split(strings.Trim(u.Path, "/"), "/")
	knownPaths := map[string]bool{
		"info": true, "hypercore": true, "evm": true,
		"nanoreth": true, "ws": true,
	}

	for _, part := range pathParts {
		if part != "" && !knownPaths[part] {
			return base + "/" + part + "/hypercore/ws", true
		}
	}

	return base + "/hypercore/ws", true
}

// extractToken extracts the API token from a QuickNode URL.
func extractToken(endpoint string) string {
	u, err := url.Parse(endpoint)
	if err != nil {
		return ""
	}

	pathParts := strings.Split(strings.Trim(u.Path, "/"), "/")
	knownPaths := map[string]bool{
		"info": true, "hypercore": true, "evm": true,
		"nanoreth": true, "ws": true, "send": true,
	}

	for _, part := range pathParts {
		if part != "" && !knownPaths[part] {
			return part
		}
	}

	return ""
}

// extractHost extracts the host (without port) from a URL.
func extractHost(endpoint string) string {
	u, err := url.Parse(endpoint)
	if err != nil {
		return ""
	}

	host := u.Host
	if strings.Contains(host, ":") {
		host = strings.Split(host, ":")[0]
	}

	return host
}
