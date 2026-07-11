package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type Config struct {
	Address      string
	RedisAddress string

	AdminEmail        string
	AdminPasswordHash string

	SessionTTL   time.Duration
	CookieSecure bool
}

func Load() (Config, error) {
	config := Config{
		Address:      envOrDefault("APP_ADDR", "127.0.0.1:8080"),
		RedisAddress: strings.TrimSpace(os.Getenv("REDIS_ADDR")),
		AdminEmail:   strings.ToLower(strings.TrimSpace(os.Getenv("ADMIN_EMAIL"))),
		SessionTTL:   12 * time.Hour,
		CookieSecure: false,
	}

	config.AdminPasswordHash = strings.TrimSpace(
		os.Getenv("ADMIN_PASSWORD_HASH"),
	)

	if config.AdminEmail == "" {
		return Config{}, fmt.Errorf(
			"ADMIN_EMAIL environment variable is required",
		)
	}

	if config.AdminPasswordHash == "" {
		return Config{}, fmt.Errorf(
			"ADMIN_PASSWORD_HASH environment variable is required",
		)
	}

	if _, err := bcrypt.Cost(
		[]byte(config.AdminPasswordHash),
	); err != nil {
		return Config{}, fmt.Errorf(
			"ADMIN_PASSWORD_HASH is not a valid bcrypt hash: %w",
			err,
		)
	}

	if value := strings.TrimSpace(
		os.Getenv("SESSION_TTL"),
	); value != "" {
		sessionTTL, err := time.ParseDuration(value)
		if err != nil {
			return Config{}, fmt.Errorf(
				"invalid SESSION_TTL: %w",
				err,
			)
		}

		if sessionTTL < 5*time.Minute {
			return Config{}, fmt.Errorf(
				"SESSION_TTL must be at least 5 minutes",
			)
		}

		config.SessionTTL = sessionTTL
	}

	if value := strings.TrimSpace(
		os.Getenv("SESSION_COOKIE_SECURE"),
	); value != "" {
		cookieSecure, err := strconv.ParseBool(value)
		if err != nil {
			return Config{}, fmt.Errorf(
				"invalid SESSION_COOKIE_SECURE: %w",
				err,
			)
		}

		config.CookieSecure = cookieSecure
	}

	return config, nil
}

func envOrDefault(
	name string,
	defaultValue string,
) string {
	value := strings.TrimSpace(os.Getenv(name))

	if value == "" {
		return defaultValue
	}

	return value
}
