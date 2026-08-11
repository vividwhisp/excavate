// Package research orchestrates the answer pipeline: search → extract →
// ground → stream, publishing progress to the message's SSE channel.
package research

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/redis/go-redis/v9"

	"excavate/internal/config"
	"excavate/internal/extract"
	"excavate/internal/llm"
	"excavate/internal/search"
	"excavate/internal/sse"
	"excavate/internal/store"
)

// Job is one unit of research work, enqueued by the HTTP handler.
type Job struct {
	ThreadID  string `json:"threadId"`
	MessageID string `json:"messageId"`
	Query     string `json:"query"`
}

// Progress stages, mirrored in the messages.status column.
const (
	StageSearching  = "searching"
	StageExtracting = "extracting"
	StageReasoning  = "reasoning"
	StageDone       = "done"
)

type Pipeline struct {
	search   search.Provider
	extract  extract.Extractor
	llm      llm.Provider
	messages *store.MessageRepo
	threads  *store.ThreadRepo
	rdb      *redis.Client
	cfg      config.AppConfig
	logger   *slog.Logger
}

func NewPipeline(
	searchP search.Provider,
	extractor extract.Extractor,
	llmP llm.Provider,
	messages *store.MessageRepo,
	threads *store.ThreadRepo,
	rdb *redis.Client,
	cfg config.AppConfig,
	logger *slog.Logger,
) *Pipeline {
	return &Pipeline{
		search:   searchP,
		extract:  extractor,
		llm:      llmP,
		messages: messages,
		threads:  threads,
		rdb:      rdb,
		cfg:      cfg,
		logger:   logger,
	}
}

// Process executes one research job and streams its progress to the SSE
// channel for the assistant message.
func (p *Pipeline) Process(ctx context.Context, job Job) {
	logger := p.logger.With("message_id", job.MessageID, "thread_id", job.ThreadID)
	if err := p.process(ctx, job, logger); err != nil {
		logger.Error("research job failed", "error", err)
		_ = p.messages.Fail(ctx, job.MessageID, err.Error())
		ev, _ := sse.NewEvent(sse.EventError, ErrorPayload{Message: err.Error()})
		_ = sse.Publish(ctx, p.rdb, job.MessageID, ev)
	}
}

func (p *Pipeline) process(ctx context.Context, job Job, logger *slog.Logger) error {
	query := strings.TrimSpace(job.Query)
	if query == "" {
		return fmt.Errorf("empty query")
	}

	_ = p.messages.SetStreaming(ctx, job.MessageID)
	defer p.threads.Touch(ctx, job.ThreadID)

	// 1) SEARCH
	if err := p.publishProgress(ctx, job.MessageID, StageSearching); err != nil {
		return err
	}
	results, err := p.search.Search(ctx, query)
	if err != nil {
		return fmt.Errorf("search: %w", err)
	}
	if len(results) > p.cfg.MaxSearchResults {
		results = results[:p.cfg.MaxSearchResults]
	}

	sources := make([]store.Source, 0, len(results))
	deliver := make([]Source, 0, len(results))
	for _, r := range results {
		sources = append(sources, store.Source{
			URL:      r.URL,
			Title:    r.Title,
			Snippet:  r.Snippet,
			Position: r.Position,
		})
		deliver = append(deliver, Source{
			URL:      r.URL,
			Title:    r.Title,
			Snippet:  r.Snippet,
			Position: r.Position,
		})
	}
	if ev, err := sse.NewEvent(sse.EventSources, SourcesPayload{Sources: deliver}); err == nil {
		_ = sse.Publish(ctx, p.rdb, job.MessageID, ev)
	}

	// 2) EXTRACT
	if err := p.publishProgress(ctx, job.MessageID, StageExtracting); err != nil {
		return err
	}
	sourceCtxs, err := p.extractPages(ctx, query, results)
	if err != nil {
		return fmt.Errorf("extract: %w", err)
	}

	// 3) REASON (stream)
	if err := p.publishProgress(ctx, job.MessageID, StageReasoning); err != nil {
		return err
	}
	content, err := p.streamAnswer(ctx, job.MessageID, query, sourceCtxs)
	if err != nil {
		return fmt.Errorf("answer: %w", err)
	}

	// 4) PERSIST
	if err := p.messages.Complete(ctx, job.MessageID, content); err != nil {
		return fmt.Errorf("persist answer: %w", err)
	}
	if err := p.messages.ReplaceSources(ctx, job.MessageID, sources); err != nil {
		return fmt.Errorf("persist sources: %w", err)
	}

	_ = p.publishProgress(ctx, job.MessageID, StageDone)
	ev, _ := sse.NewEvent(sse.EventDone, DonePayload{})
	_ = sse.Publish(ctx, p.rdb, job.MessageID, ev)
	return nil
}

func (p *Pipeline) extractPages(ctx context.Context, query string, results []search.Result) ([]llm.SourceContext, error) {
	n := p.cfg.MaxExtractPages
	if len(results) < n {
		n = len(results)
	}

	sourceCtxs := make([]llm.SourceContext, 0, n)
	for i := 0; i < n; i++ {
		page, err := p.extract.Extract(ctx, results[i].URL)
		if err != nil {
			p.logger.Warn("extract failed, skipping page", "url", results[i].URL, "error", err)
			continue
		}
		sourceCtxs = append(sourceCtxs, llm.SourceContext{
			URL:      page.URL,
			Title:    results[i].Title,
			Snippet:  results[i].Snippet,
			Content:  truncate(page.Content, p.cfg.MaxContextChars/int(n)),
			Position: i + 1,
		})
	}
	return sourceCtxs, nil
}

func (p *Pipeline) streamAnswer(ctx context.Context, messageID, query string, sourceCtxs []llm.SourceContext) (string, error) {
	deltas, err := p.llm.StreamAnswer(ctx, llm.Prompt{Query: query, Sources: sourceCtxs})
	if err != nil {
		return "", err
	}

	var content strings.Builder
	for d := range deltas {
		content.WriteString(d.Text)
		ev, err := sse.NewEvent(sse.EventDelta, DeltaPayload{Text: d.Text})
		if err != nil {
			continue
		}
		if err := sse.Publish(ctx, p.rdb, messageID, ev); err != nil {
			return "", err
		}
	}
	return content.String(), nil
}

func (p *Pipeline) publishProgress(ctx context.Context, messageID, stage string) error {
	ev, err := sse.NewEvent(sse.EventProgress, ProgressPayload{Stage: stage})
	if err != nil {
		return err
	}
	return sse.Publish(ctx, p.rdb, messageID, ev)
}

func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if len(s) <= max {
		return s
	}
	return s[:max]
}
