package cache

import (
	"context"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// QueryCache caches grounded answers for identical queries to stretch
// the free Google CSE budget (100 searches/day).
type QueryCache struct {
	rdb *redis.Client
	ttl time.Duration
}

type CachedAnswer struct {
	Content string
	Sources []CachedSource
}

type CachedSource struct {
	URL      string
	Title    string
	Snippet  string
	Position int
}

func NewQueryCache(rdb *redis.Client, ttl time.Duration) *QueryCache {
	return &QueryCache{rdb: rdb, ttl: ttl}
}

func queryKey(q string) string {
	return "answer:" + strings.ToLower(strings.TrimSpace(q))
}

func (c *QueryCache) Get(ctx context.Context, query string) (*CachedAnswer, error) {
	var a CachedAnswer
	if err := c.rdb.Get(ctx, queryKey(query)).Scan(&a); err != nil {
		return nil, err
	}
	return &a, nil
}

func (c *QueryCache) Set(ctx context.Context, query string, answer *CachedAnswer) error {
	return c.rdb.Set(ctx, queryKey(query), answer, c.ttl).Err()
}