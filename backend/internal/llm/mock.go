package llm

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Mock streams a canned, citation-backed markdown answer so the full pipeline
// (SSE → persistence → UI) can be tested without an API key. It still
// references the real source positions so citation badges have valid data.
type Mock struct{}

func NewMock() *Mock { return &Mock{} }

func (m *Mock) StreamAnswer(ctx context.Context, prompt Prompt) (chan Delta, error) {
	answer := m.buildAnswer(prompt)
	ch := make(chan Delta)

	go func() {
		defer close(ch)
		for _, part := range splitParts(answer) {
			select {
			case <-ctx.Done():
				return
			case <-time.After(60 * time.Millisecond):
				ch <- Delta{Text: part}
			}
		}
	}()
	return ch, nil
}

func (m *Mock) buildAnswer(prompt Prompt) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## About %q\n\n", prompt.Query)

	if len(prompt.Sources) > 0 {
		s := prompt.Sources[0]
		fmt.Fprintf(&b,
			"Based on the available sources, %s is best understood through a few key points. "+
				"According to %s, %s[1]\n\n",
			prompt.Query, s.Title, s.Content)
	}

	b.WriteString("Key findings:\n\n")

	for i, s := range prompt.Sources {
		n := i + 1
		fmt.Fprintf(&b, "- %s reports: %s[%d]\n", s.Title, s.Snippet, n)
	}

	if len(prompt.Sources) > 1 {
		fmt.Fprintf(&b,
			"\nTaken together, sources agree that %s is a well-studied topic, "+
				"with %s and %s providing the most directly relevant detail. "+
				"Further reading is available in the source list below.\n\n",
			prompt.Query,
			citeLabel(prompt.Sources, 1), citeLabel(prompt.Sources, 2))
	}

	b.WriteString("## Sources\n\n")
	for i, s := range prompt.Sources {
		fmt.Fprintf(&b, "[%d] [%s](%s)\n", i+1, s.Title, s.URL)
	}
	return b.String()
}

func citeLabel(sources []SourceContext, pos int) string {
	if pos <= len(sources) {
		return sources[pos-1].Title
	}
	return "the sources"
}

// splitParts chunks the answer roughly per sentence so the stream looks alive.
func splitParts(answer string) []string {
	parts := []string{}
	var cur strings.Builder
	for _, word := range strings.Fields(answer) {
		cur.WriteString(word)
		cur.WriteString(" ")
		if len(cur.String()) > 0 && (strings.HasSuffix(word, ".") || strings.HasSuffix(word, ":") || len(cur.String()) > 120) {
			parts = append(parts, cur.String())
			cur.Reset()
		}
	}
	if cur.Len() > 0 {
		parts = append(parts, cur.String())
	}
	return parts
}

var _ Provider = (*Mock)(nil)
