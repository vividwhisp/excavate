// Package llm defines the answer-generation provider abstraction. Real
// providers (e.g. Google Gemini) implement Provider (M4).
package llm

import "context"

// SourceContext is a grounded source handed to the model.
type SourceContext struct {
	URL      string
	Title    string
	Snippet  string
	Content  string
	Position int
}

// Prompt is everything the model needs to answer: the query plus grounded
// context from the search + extract stages.
type Prompt struct {
	Query   string
	Sources []SourceContext
}

// Delta is one streaming chunk of the generated answer.
type Delta struct {
	Text string
}

type Provider interface {
	// StreamAnswer streams the answer as a sequence of deltas.
	// The returned channel is closed when streaming finishes.
	StreamAnswer(ctx context.Context, prompt Prompt) (chan Delta, error)
}
