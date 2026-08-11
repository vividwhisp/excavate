// Package search defines the search provider abstraction used by the
// research pipeline. Real providers (e.g. Google CSE) implement Provider.
package search

import "context"

type Result struct {
	URL     string `json:"url"`
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
	Position int   `json:"position"`
}

type Provider interface {
	// Search returns ranked results for the query.
	Search(ctx context.Context, query string) ([]Result, error)
}
