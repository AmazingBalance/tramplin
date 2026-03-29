package config

import (
	"net"
	"net/url"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr                string
	TokenSecret             string
	AccessTokenTTL          time.Duration
	RefreshTokenTTL         time.Duration
	AccessCookieName        string
	RefreshCookieName       string
	CORSAllowOrigin         string
	CookieSecure            bool
	CookieDomain            string
	DefaultAccessTokenTTL   int
	DatabaseURL             string
	DatabaseConnectTimeout  time.Duration
	DatabasePingTimeout     time.Duration
	DatabaseMaxConns        int32
	DatabaseMinConns        int32
	DatabaseMaxConnLifetime time.Duration
	DatabaseMaxConnIdleTime time.Duration
}

func Load() Config {
	databaseURL := os.Getenv("TRAMPLIN_DATABASE_URL")
	if databaseURL == "" {
		databaseURL = buildDatabaseURL()
	}

	return Config{
		HTTPAddr:                env("TRAMPLIN_HTTP_ADDR", ":8080"),
		TokenSecret:             env("TRAMPLIN_TOKEN_SECRET", "dev-secret-change-me"),
		AccessTokenTTL:          15 * time.Minute,
		RefreshTokenTTL:         7 * 24 * time.Hour,
		AccessCookieName:        env("TRAMPLIN_ACCESS_COOKIE", "tramplin_access_token"),
		RefreshCookieName:       env("TRAMPLIN_REFRESH_COOKIE", "tramplin_refresh_token"),
		CORSAllowOrigin:         env("TRAMPLIN_CORS_ALLOW_ORIGIN", ""),
		CookieSecure:            env("TRAMPLIN_COOKIE_SECURE", "") == "true",
		CookieDomain:            env("TRAMPLIN_COOKIE_DOMAIN", ""),
		DefaultAccessTokenTTL:   900,
		DatabaseURL:             databaseURL,
		DatabaseConnectTimeout:  envDuration("TRAMPLIN_DATABASE_CONNECT_TIMEOUT", 5*time.Second),
		DatabasePingTimeout:     envDuration("TRAMPLIN_DATABASE_PING_TIMEOUT", 2*time.Second),
		DatabaseMaxConns:        envInt32("TRAMPLIN_DATABASE_MAX_CONNS", 10),
		DatabaseMinConns:        envInt32("TRAMPLIN_DATABASE_MIN_CONNS", 0),
		DatabaseMaxConnLifetime: envDuration("TRAMPLIN_DATABASE_MAX_CONN_LIFETIME", time.Hour),
		DatabaseMaxConnIdleTime: envDuration("TRAMPLIN_DATABASE_MAX_CONN_IDLE_TIME", 15*time.Minute),
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	value, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return value
}

func envInt32(key string, fallback int32) int32 {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseInt(raw, 10, 32)
	if err != nil {
		return fallback
	}
	return int32(value)
}

func buildDatabaseURL() string {
	user := os.Getenv("TRAMPLIN_DATABASE_USER")
	if user == "" {
		user = "postgres"
	}
	password := os.Getenv("TRAMPLIN_DATABASE_PASSWORD")
	host := env("TRAMPLIN_DATABASE_HOST", "127.0.0.1")
	port := env("TRAMPLIN_DATABASE_PORT", "5432")
	name := env("TRAMPLIN_DATABASE_NAME", "tramplin")
	sslMode := env("TRAMPLIN_DATABASE_SSLMODE", "disable")

	databaseURL := &url.URL{
		Scheme: "postgres",
		Host:   net.JoinHostPort(host, port),
		Path:   "/" + name,
	}

	if password != "" {
		databaseURL.User = url.UserPassword(user, password)
	} else if user != "" {
		databaseURL.User = url.User(user)
	}

	query := databaseURL.Query()
	query.Set("sslmode", sslMode)
	databaseURL.RawQuery = query.Encode()

	return databaseURL.String()
}
