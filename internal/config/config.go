package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port   string
	AppEnv string
	AppURL string

	JWTAccessSecret  string
	JWTRefreshSecret string
	JWTAccessExpiry  time.Duration
	JWTRefreshExpiry time.Duration

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	RedisHost     string
	RedisPort     string
	RedisPassword string

	OpenAIAPIKey          string
	OpenAIModel           string
	OpenAIBaseURL         string
	OpenAIReasoningEffort string

	FonnteToken        string
	FonnteWebhookToken string
	FonnteDeviceID     string

	AdminDefaultUsername string
	AdminDefaultPassword string

	BcryptCost int
}

func Load() Config {
	return Config{
		Port:   env("PORT", "8080"),
		AppEnv: env("APP_ENV", "development"),
		AppURL: env("APP_URL", "http://localhost:8080"),

		JWTAccessSecret:  env("JWT_ACCESS_SECRET", "dev_access_secret_change_me"),
		JWTRefreshSecret: env("JWT_REFRESH_SECRET", "dev_refresh_secret_change_me"),
		JWTAccessExpiry:  envDuration("JWT_ACCESS_EXPIRY", 15*time.Minute),
		JWTRefreshExpiry: envDuration("JWT_REFRESH_EXPIRY", 168*time.Hour),

		DBHost:     env("DB_HOST", "localhost"),
		DBPort:     env("DB_PORT", "5432"),
		DBUser:     env("DB_USER", "postgres"),
		DBPassword: env("DB_PASSWORD", "postgres"),
		DBName:     env("DB_NAME", "finance_tracker"),
		DBSSLMode:  env("DB_SSLMODE", "disable"),

		RedisHost:     env("REDIS_HOST", "localhost"),
		RedisPort:     env("REDIS_PORT", "6379"),
		RedisPassword: env("REDIS_PASSWORD", ""),

		OpenAIAPIKey:          envAny([]string{"KIE_AI_API_KEY", "OPENAI_API_KEY"}, ""),
		OpenAIModel:           envAny([]string{"KIE_AI_MODEL", "OPENAI_MODEL"}, "gpt-4o-mini"),
		OpenAIBaseURL:         envAny([]string{"KIE_AI_BASE_URL", "OPENAI_BASE_URL"}, "https://api.kie.ai"),
		OpenAIReasoningEffort: envAny([]string{"KIE_AI_REASONING_EFFORT", "OPENAI_REASONING_EFFORT"}, "low"),

		FonnteToken:        env("FONNTE_TOKEN", ""),
		FonnteWebhookToken: env("FONNTE_WEBHOOK_TOKEN", ""),
		FonnteDeviceID:     env("FONNTE_DEVICE_ID", ""),

		AdminDefaultUsername: env("ADMIN_DEFAULT_USERNAME", "admin"),
		AdminDefaultPassword: env("ADMIN_DEFAULT_PASSWORD", "admin12345"),

		BcryptCost: envInt("BCRYPT_COST", 12),
	}
}

func (c Config) DatabaseURL() string {
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.DBUser, c.DBPassword),
		Host:   c.DBHost + ":" + c.DBPort,
		Path:   c.DBName,
	}
	q := u.Query()
	q.Set("sslmode", c.DBSSLMode)
	u.RawQuery = q.Encode()
	return u.String()
}

func (c Config) MigrationDatabaseURL() string {
	return c.DatabaseURL()
}

func (c Config) RedisAddr() string {
	return fmt.Sprintf("%s:%s", c.RedisHost, c.RedisPort)
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envAny(keys []string, fallback string) string {
	for _, key := range keys {
		if value := os.Getenv(key); value != "" {
			return value
		}
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
