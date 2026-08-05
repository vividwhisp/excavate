package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// SessionStore stores opaque session tokens with user ID + expiry.
type SessionStore struct {
	rdb *redis.Client
	ttl time.Duration
}

func NewSessionStore(rdb *redis.Client, ttl time.Duration) *SessionStore {
	return &SessionStore{rdb: rdb, ttl: ttl}
}

func sessionKey(token string) string { return "session:" + token }

// Create stores the token, refreshing its TTL each access.
func (s *SessionStore) Create(ctx context.Context, token, userID string) error {
	return s.rdb.Set(ctx, sessionKey(token), userID, s.ttl).Err()
}

func (s *SessionStore) Get(ctx context.Context, token string) (string, error) {
	userID, err := s.rdb.Get(ctx, sessionKey(token)).Result()
	if err != nil {
		return "", err
	}
	// Sliding expiry.
	s.rdb.Expire(ctx, sessionKey(token), s.ttl)
	return userID, nil
}

func (s *SessionStore) Delete(ctx context.Context, token string) error {
	return s.rdb.Del(ctx, sessionKey(token)).Err()
}

func (s *SessionStore) TTL() time.Duration { return s.ttl }