package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const defaultTavilySearchURL = "https://api.tavily.com/search"

type TavilyClient interface {
	Search(ctx context.Context, query string, topK int) ([]SearchResult, error)
}

type HTTPClient struct {
	apiKey     string
	searchURL  string
	httpClient *http.Client
}

func NewHTTPClient(cfg Config, httpClient *http.Client) *HTTPClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 20 * time.Second}
	}
	return &HTTPClient{
		apiKey:     strings.TrimSpace(cfg.APIKey),
		searchURL:  normalizeSearchURL(cfg.BaseURL),
		httpClient: httpClient,
	}
}

func (c *HTTPClient) Search(ctx context.Context, query string, topK int) ([]SearchResult, error) {
	if topK <= 0 {
		topK = 5
	}
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("query is empty")
	}

	body, err := json.Marshal(map[string]any{
		"query":       query,
		"max_results": topK,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.searchURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("tavily request failed: status=%d", resp.StatusCode)
	}

	var payload struct {
		Results []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Content string `json:"content"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	results := make([]SearchResult, 0, len(payload.Results))
	for _, item := range payload.Results {
		results = append(results, SearchResult{
			Title:   item.Title,
			URL:     item.URL,
			Snippet: item.Content,
		})
	}
	return results, nil
}

func normalizeSearchURL(baseURL string) string {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return defaultTavilySearchURL
	}
	if strings.HasSuffix(baseURL, "/search") {
		return baseURL
	}
	return strings.TrimRight(baseURL, "/") + "/search"
}
