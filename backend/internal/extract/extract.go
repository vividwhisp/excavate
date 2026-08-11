// Package extract fetches web pages and converts them to plain text for
// grounding the answer. Real implementations fetch + strip HTML (M4).
package extract

import "context"

type Page struct {
	URL     string `json:"url"`
	Title   string `json:"title"`
	Content string `json:"content"` // plain text body
}

type Extractor interface {
	// Extract returns the plain-text body of the page at url.
	Extract(ctx context.Context, url string) (Page, error)
}
