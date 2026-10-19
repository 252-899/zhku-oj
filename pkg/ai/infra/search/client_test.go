package search

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newTestHTTPClient(fn roundTripFunc) *http.Client {
	return &http.Client{Transport: fn}
}

func TestHTTPClientSearchSuccess(t *testing.T) {
	client := NewHTTPClient(Config{APIKey: "test-key", BaseURL: "https://tavily.test"}, newTestHTTPClient(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("unexpected Authorization header: %s", r.Header.Get("Authorization"))
		}
		if r.URL.String() != "https://tavily.test/search" {
			t.Fatalf("unexpected URL: %s", r.URL.String())
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body error: %v", err)
		}
		if body["query"] != "golang" {
			t.Fatalf("unexpected query: %#v", body["query"])
		}
		if int(body["max_results"].(float64)) != 3 {
			t.Fatalf("unexpected max_results: %#v", body["max_results"])
		}

		payload, _ := json.Marshal(map[string]any{
			"results": []map[string]any{
				{
					"title":   "Go",
					"url":     "https://go.dev",
					"content": "The Go Programming Language",
				},
			},
		})
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(bytes.NewReader(payload)),
		}, nil
	}))
	results, err := client.Search(context.Background(), "golang", 3)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Snippet != "The Go Programming Language" {
		t.Fatalf("unexpected snippet: %q", results[0].Snippet)
	}
}

func TestHTTPClientSearchHandlesEmptyResults(t *testing.T) {
	client := NewHTTPClient(Config{APIKey: "test-key", BaseURL: "https://tavily.test"}, newTestHTTPClient(func(r *http.Request) (*http.Response, error) {
		payload, _ := json.Marshal(map[string]any{"results": []map[string]any{}})
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(bytes.NewReader(payload)),
		}, nil
	}))
	results, err := client.Search(context.Background(), "golang", 0)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestHTTPClientSearchHandlesUpstreamError(t *testing.T) {
	client := NewHTTPClient(Config{APIKey: "test-key", BaseURL: "https://tavily.test"}, newTestHTTPClient(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusBadGateway,
			Header:     make(http.Header),
			Body:       io.NopCloser(bytes.NewBufferString("boom")),
		}, nil
	}))
	if _, err := client.Search(context.Background(), "golang", 2); err == nil {
		t.Fatal("expected upstream error")
	}
}

func TestHTTPClientSearchHandlesTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	httpClient := newTestHTTPClient(func(r *http.Request) (*http.Response, error) {
		select {
		case <-time.After(100 * time.Millisecond):
			payload, _ := json.Marshal(map[string]any{"results": []map[string]any{}})
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(bytes.NewReader(payload)),
			}, nil
		case <-r.Context().Done():
			return nil, r.Context().Err()
		}
	})

	client := NewHTTPClient(Config{APIKey: "test-key", BaseURL: "https://tavily.test"}, httpClient)
	if _, err := client.Search(ctx, "golang", 2); err == nil {
		t.Fatal("expected timeout error")
	}
}
