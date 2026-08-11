package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Env      string
	Port     string
	Debug    bool
	BaseURL  string
	Database string
	Redis    RedisConfig
	Session  SessionConfig
	CORS     []string
	App      AppConfig
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type SessionConfig struct {
	Secret string
	TTL    time.Duration
	Name   string
}

type AppConfig struct {
	// Limits / tunables for the research pipeline.
	MaxSearchResults  int
	MaxExtractPages   int
	MaxContextChars   int
	QueryCacheTTL     time.Duration
	RateLimitPerHour  int
}

func Load() Config {
	return Config{
		Env:      getEnv("APP_ENV", "development"),
		Port:     getEnv("PORT", "8080"),
		Debug:    getEnvBool("DEBUG", false),
		BaseURL:  getEnv("BASE_URL", "http://localhost:8080"),
		Database: getEnv("DATABASE_URL", "postgres://excavate:excavate@localhost:5432/excavate?sslmode=disable"),
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "127.0.0.1:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		Session: SessionConfig{
			Secret: getEnv("SESSION_SECRET", "dev-session-secret-change-me"),
			TTL:    time.Duration(getEnvInt("SESSION_TTL_HOURS", 168)) * time.Hour,
			Name:   getEnv("SESSION_COOKIE", "excavate_session"),
		},
		CORS: []string{getEnv("CORS_ORIGIN", "http://localhost:5173")},
		App: AppConfig{
			MaxSearchResults: getEnvInt("MAX_SEARCH_RESULTS", 8),
			MaxExtractPages:  getEnvInt("MAX_EXTRACT_PAGES", 5),
			MaxContextChars:  getEnvInt("MAX_CONTEXT_CHARS", 24000),
			QueryCacheTTL:    time.Duration(getEnvInt("QUERY_CACHE_TTL_MINUTES", 60)) * time.Minute,
			RateLimitPerHour: getEnvInt("RATE_LIMIT_PER_HOUR", 60),
		},
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}
