package research

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

// QueueKey is the Redis list holding pending research jobs.
const QueueKey = "jobs:research"

// --- SSE payloads (consumed by the React frontend in M5) ---

type ProgressPayload struct {
	Stage string `json:"stage"`
}

type SourcesPayload struct {
	Sources []Source `json:"sources"`
}

// Source is the JSON shape delivered to the browser for the sources bar.
type Source struct {
	URL      string `json:"url"`
	Title    string `json:"title"`
	Snippet  string `json:"snippet"`
	Position int    `json:"position"`
}

type DeltaPayload struct {
	Text string `json:"text"`
}

type DonePayload struct{}

type ErrorPayload struct {
	Message string `json:"message"`
}

// Enqueue pushes a job onto the research queue.
func Enqueue(ctx context.Context, rdb *redis.Client, job Job) error {
	b, err := json.Marshal(job)
	if err != nil {
		return err
	}
	return rdb.LPush(ctx, QueueKey, b).Err()
}

// Worker consumes jobs from the queue and runs them through the pipeline.
// Run blocks until ctx is cancelled; intended to run as a background goroutine.
func (p *Pipeline) Worker(ctx context.Context, workers int) {
	logger := p.logger.With("component", "research-worker")
	for i := 0; i < workers; i++ {
		go func(id int) {
			logger.Info("worker started", "id", id)
			for {
				if err := p.consume(ctx); err != nil {
					if ctx.Err() != nil {
						logger.Info("worker stopping", "id", id)
						return
					}
					logger.Warn("worker error", "id", id, "error", err)
					select {
					case <-ctx.Done():
						return
					case <-time.After(500 * time.Millisecond):
					}
				}
			}
		}(i)
	}
}

func (p *Pipeline) consume(ctx context.Context) error {
	// BLPOP blocks until a job arrives; it honours ctx via the Redis timeout
	// so we check cancellation regularly.
	res, err := p.rdb.BLPop(ctx, 3*time.Second, QueueKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil // timeout, keep looping
		}
		return err
	}

	var job Job
	if err := json.Unmarshal([]byte(res[1]), &job); err != nil {
		return err
	}
	p.Process(ctx, job)
	return nil
}
