package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const geminiEndpoint = "https://generativelanguage.googleapis.com/v1beta"

// Gemini is a llm.Provider backed by Google's Gemini API. It uses the plain
// generateContent endpoint (works on the free tier); Google Search grounding
// can be enabled later by adding the googleSearch tool to the request body.
type Gemini struct {
	apiKey string
	model  string
	client *http.Client
}

func NewGemini(apiKey, model string) *Gemini {
	return &Gemini{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{Timeout: 120 * time.Second},
	}
}

// --- wire format (subset we need) ---

type geminiRequest struct {
	SystemInstruction *geminiContent `json:"systemInstruction,omitempty"`
	Contents          []geminiContent `json:"contents"`
	GenerationConfig  *geminiGenConfig `json:"generationConfig,omitempty"`
}

type geminiContent struct {
	Role  string        `json:"role,omitempty"`
	Parts []geminiPart  `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenConfig struct {
	MaxOutputTokens int `json:"maxOutputTokens"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []geminiPart `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
}

func (g *Gemini) StreamAnswer(ctx context.Context, prompt Prompt) (chan Delta, error) {
	reqBody := geminiRequest{
		SystemInstruction: &geminiContent{
			Parts: []geminiPart{{Text: systemPrompt}},
		},
		Contents: []geminiContent{
			{Role: "user", Parts: []geminiPart{{Text: buildUserPrompt(prompt)}}},
		},
		GenerationConfig: &geminiGenConfig{MaxOutputTokens: 4096},
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/models/%s:streamGenerateContent?alt=sse&key=%s", geminiEndpoint, g.model, g.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		resp.Body.Close()
		return nil, fmt.Errorf("gemini: status %d: %s", resp.StatusCode, truncate(string(raw), 300))
	}

	ch := make(chan Delta)
	go func() {
		defer close(ch)
		defer resp.Body.Close()
		g.scanStream(ctx, resp.Body, ch)
	}()
	return ch, nil
}

func (g *Gemini) scanStream(ctx context.Context, body io.Reader, ch chan<- Delta) {
	sc := bufio.NewScanner(body)
	// Thought signatures are large; default scanner limit (64KiB) is too small.
	sc.Buffer(make([]byte, 256*1024), 16*1024*1024)

	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}

		var gr geminiResponse
		if err := json.Unmarshal([]byte(data), &gr); err != nil {
			continue
		}
		for _, c := range gr.Candidates {
			for _, p := range c.Content.Parts {
				text := strings.TrimSpace(p.Text)
				if text == "" {
					continue
				}
				select {
				case ch <- Delta{Text: p.Text}:
				case <-ctx.Done():
					return
				}
			}
		}
	}
}

const systemPrompt = `You are Excavate, a precise research assistant. Answer the user's question using ONLY the provided sources, which are listed as [1], [2], etc. Cite each claim with the source number(s) in brackets, like [1] or [2][3]. If the sources don't answer the question, say so instead of guessing. Write a clear markdown answer with a short summary up top, then the detailed findings. End with a "## Sources" section listing each source as "[n] title (url)".`

func buildUserPrompt(prompt Prompt) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Question: %s\n\nSources:\n", prompt.Query)
	for i, s := range prompt.Sources {
		n := i + 1
		fmt.Fprintf(&b, "[%d] %s\n    URL: %s\n    %s\n", n, s.Title, s.URL, s.Snippet)
		if s.Content != "" {
			fmt.Fprintf(&b, "    %s\n", truncate(s.Content, 1500))
		}
	}
	return b.String()
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}

var _ Provider = (*Gemini)(nil)