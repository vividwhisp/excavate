package config

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Env          string
	Port         string
	Debug        bool
	BaseURL      string
	Database     string
	Redis        RedisConfig
	Session      SessionConfig
	CORS         []string
	App          AppConfig
	Gemini       GeminiConfig
	Tavily       TavilyConfig
	ResearchMode string
}

// GeminiConfig holds credentials for the Google Gemini API (free tier).
type GeminiConfig struct {
	APIKey string
	Model  string
}

// TavilyConfig holds the API key for the Tavily Search API (free tier:
// 1,000 credits/month, no credit card required).
type TavilyConfig struct {
	APIKey string
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
	loadDotEnv()
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
		Gemini: GeminiConfig{
			APIKey: getEnv("GEMINI_API_KEY", ""),
			Model:  getEnv("GEMINI_MODEL", "gemini-3.5-flash"),
		},
		Tavily: TavilyConfig{
			APIKey: getEnv("TAVILY_API_KEY", ""),
		},
		ResearchMode: getEnv("RESEARCH_MODE", "auto"),
	}
}

// loadDotEnv reads KEY=VALUE pairs from a .env file in the working directory.
// Existing environment variables always win, so tests can override by setting
// variables before the process starts. It is deliberately dependency-free.
func loadDotEnv() {
	f, err := os.Open(".env")
	if err != nil {
		return
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		eq := strings.Index(line, "=")
		if eq <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		val := strings.TrimSpace(line[eq+1:])
		val = strings.Trim(val, `"'`)
		if _, ok := os.LookupEnv(key); !ok {
			_ = os.Setenv(key, val)
		}
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
