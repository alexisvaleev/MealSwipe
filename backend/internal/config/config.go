// Package config parses environment configuration (SPEC §10).
package config

import (
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config is the resolved application configuration.
type Config struct {
	AppName        string
	DatabaseURL    string
	HTTPAddr       string
	CookieSecure   bool
	SessionTTL     time.Duration
	SeedOnStart    bool
	AllowedOrigins []string
	LogLevel       slog.Level
	MealDBAPIKey   string
}

// Load reads configuration from the environment.
func Load() (Config, error) {
	c := Config{
		AppName:      str("APP_NAME", "MealSwipe"),
		HTTPAddr:     str("HTTP_ADDR", ":8080"),
		CookieSecure: boolean("COOKIE_SECURE", false),
		SessionTTL:   time.Duration(integer("SESSION_TTL_HOURS", 720)) * time.Hour,
		SeedOnStart:  boolean("SEED_ON_START", true),
		MealDBAPIKey: str("MEALDB_API_KEY", "1"),
	}

	switch strings.ToLower(str("LOG_LEVEL", "info")) {
	case "debug":
		c.LogLevel = slog.LevelDebug
	case "info":
		c.LogLevel = slog.LevelInfo
	case "warn", "warning":
		c.LogLevel = slog.LevelWarn
	case "error":
		c.LogLevel = slog.LevelError
	default:
		return c, fmt.Errorf("config: unknown LOG_LEVEL %q", os.Getenv("LOG_LEVEL"))
	}

	if v := str("ALLOWED_ORIGINS", ""); v != "" {
		for _, o := range strings.Split(v, ",") {
			if o = strings.TrimSpace(o); o != "" {
				c.AllowedOrigins = append(c.AllowedOrigins, o)
			}
		}
	}

	c.DatabaseURL = os.Getenv("DATABASE_URL")
	if c.DatabaseURL == "" {
		// Compose-style fallback: build the DSN from POSTGRES_* parts.
		user := str("POSTGRES_USER", "mealswipe")
		pass := str("POSTGRES_PASSWORD", "mealswipe")
		name := str("POSTGRES_DB", "mealswipe")
		host := str("POSTGRES_HOST", "localhost")
		port := str("POSTGRES_PORT", "5432")
		u := &url.URL{
			Scheme:   "postgres",
			User:     url.UserPassword(user, pass),
			Host:     host + ":" + port,
			Path:     "/" + name,
			RawQuery: "sslmode=disable",
		}
		c.DatabaseURL = u.String()
	}
	return c, nil
}

func str(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func boolean(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

func integer(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return def
	}
	return n
}
