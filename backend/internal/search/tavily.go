package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const tavilyEndpoint = "https://api.tavily.com/search"

// Tavily is a search.Provider backed by the Tavily Search API
// (free tier: 1,000 credits/month, no credit card required).
type Tavily struct {
	apiKey string
	client *http.Client
}

func NewTavily(apiKey string) *Tavily {
	return &Tavily{
		apiKey: apiKey,
		client: &http.Client{Timeout: 20 * time.Second},
	}
}

// tavilyResult mirrors one entry of Tavily's results array.
type tavilyResult struct {
	Title   string  `json:"title"`
	URL     string  `json:"url"`
	Content string  `json:"content"`
	Score   float64 `json:"score"`
}

// tavilyResponse mirrors the Tavily /search response body.
type tavilyResponse struct {
	Query   string          `json:"query"`
	Results []tavilyResult  `json:"results"`
	Answer  string          `json:"answer"`
}

func (t *Tavily) Search(ctx context.Context, query string) ([]Result, error) {
	body, err := json.Marshal(map[string]any{
		"query":          query,
		"max_results":    8,
		"search_depth":   "basic",
		"include_answer": false,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tavilyEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+t.apiKey)

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tavily search: status %d: %s", resp.StatusCode, truncate(string(raw), 300))
	}

	var tr tavilyResponse
	if err := json.Unmarshal(raw, &tr); err != nil {
		return nil, fmt.Errorf("tavily search: decode: %w", err)
	}

	results := make([]Result, 0, len(tr.Results))
	for i, r := range tr.Results {
		if r.URL == "" {
			continue
		}
		results = append(results, Result{
			URL:      r.URL,
			Title:    r.Title,
			Snippet:  truncate(r.Content, 400),
			Position: i + 1,
		})
	}
	return results, nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}

var _ Provider = (*Tavily)(nil)