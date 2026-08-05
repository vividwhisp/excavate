package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimiter implements a simple sliding-window limiter using sorted sets.
type RateLimiter struct {
	rdb *redis.Client
}

func NewRateLimiter(rdb *redis.Client) *RateLimiter {
	return &RateLimiter{rdb: rdb}
}

// Allow reports whether key may proceed, given max events per window.
func (l *RateLimiter) Allow(ctx context.Context, key string, max int, window time.Duration) (bool, error) {
	now := float64(time.Now().UnixNano()) / 1e9
	start := now - window.Seconds()
	zkey := fmt.Sprintf("rl:%s", key)

	pipe := l.rdb.TxPipeline()
	pipe.ZRemRangeByScore(ctx, zkey, "0", fmt.Sprintf("%f", start))
	pipe.ZAdd(ctx, zkey, redis.Z{Score: now, Member: fmt.Sprintf("%d", time.Now().UnixNano())})
	pipe.Expire(ctx, zkey, window)
	countCmd := pipe.ZCard(ctx, zkey)

	if _, err := pipe.Exec(ctx); err != nil {
		return false, err
	}
	count := countCmd.Val()
	return count <= int64(max), nil
}