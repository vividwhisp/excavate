package search

import (
	"context"
	"time"
)

// Mock is a deterministic search provider used until real providers are wired
// up. It returns stable, clearly-fake results so the pipeline can be tested
// end-to-end without any API keys.
type Mock struct{}

func NewMock() *Mock { return &Mock{} }

func (m *Mock) Search(ctx context.Context, query string) ([]Result, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(300 * time.Millisecond):
	}

	titles := []string{
		"Introduction to Go programming",
		"Building REST APIs with Go",
		"PostgreSQL documentation",
		"Redis in-memory data store",
	}
	return []Result{
		{URL: "https://example.com/go-intro", Title: titles[0], Snippet: "A beginner's guide to the Go programming language.", Position: 1},
		{URL: "https://example.com/go-apis", Title: titles[1], Snippet: "How to build production REST APIs using chi.", Position: 2},
		{URL: "https://example.com/postgres-docs", Title: titles[2], Snippet: "Official PostgreSQL manual and reference.", Position: 3},
		{URL: "https://example.com/redis-docs", Title: titles[3], Snippet: "The open-source, in-memory data structure store.", Position: 4},
	}[0:min(4, 4)], nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

var _ Provider = (*Mock)(nil)
