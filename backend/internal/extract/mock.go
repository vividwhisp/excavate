package extract

import (
	"context"
	"fmt"
	"time"
)

// Mock returns a canned page body per URL so the pipeline can be exercised
// without network access or API keys.
type Mock struct{}

func NewMock() *Mock { return &Mock{} }

var bodies = map[string]string{
	"https://example.com/go-intro": "Go is a statically typed, compiled programming language designed at Google. It is known for its simplicity, fast compilation, built-in concurrency and garbage collection.",
	"https://example.com/go-apis":  "Building REST APIs in Go with the chi router gives you a composable, idiomatic HTTP layer. Middleware handles logging, auth and recovery, and handlers stay small and testable.",
	"https://example.com/postgres-docs": "PostgreSQL is a powerful, open source object-relational database system with strong reliability, feature robustness, and performance.",
	"https://example.com/redis-docs": "Redis is an in-memory data structure store used as a database, cache and message broker. It supports data structures such as strings, hashes and lists.",
}

func (m *Mock) Extract(ctx context.Context, url string) (Page, error) {
	select {
	case <-ctx.Done():
		return Page{}, ctx.Err()
	case <-time.After(400 * time.Millisecond):
	}

	body, ok := bodies[url]
	if !ok {
		body = fmt.Sprintf("This page (%s) describes background material relevant to the query.", url)
	}
	return Page{URL: url, Title: url, Content: body}, nil
}

var _ Extractor = (*Mock)(nil)
