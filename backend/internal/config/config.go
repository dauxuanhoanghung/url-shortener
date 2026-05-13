package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Server   ServerConfig
	Postgres PostgresConfig
	Redis    RedisConfig
	JWT      JWTConfig
	Stripe   StripeConfig
}

type ServerConfig struct {
	Port            string
	Mode            string
	BaseURL         string
	FrontendBaseURL string // used in verification / reset-password emails
}

type PostgresConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DB       string
}

func (p PostgresConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		p.User, p.Password, p.Host, p.Port, p.DB,
	)
}

type RedisConfig struct {
	Host string
	Port string
}

func (r RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%s", r.Host, r.Port)
}

type JWTConfig struct {
	Secret string
}

type StripeConfig struct {
	SecretKey     string
	WebhookSecret string
}

func Load() (*Config, error) {
	_ = godotenv.Load("../../.env")

	cfg := &Config{
		Server: ServerConfig{
			Port:            getEnv("SERVER_PORT", "8080"),
			Mode:            getEnv("GIN_MODE", "debug"),
			BaseURL:         getEnv("BASE_URL", "http://localhost:8080"),
			FrontendBaseURL: getEnv("FRONTEND_BASE_URL", "http://localhost:5173"),
		},
		Postgres: PostgresConfig{
			Host:     getEnv("POSTGRES_HOST", "localhost"),
			Port:     getEnv("POSTGRES_PORT", "5432"),
			User:     getEnv("POSTGRES_USER", "urlshortener"),
			Password: getEnv("POSTGRES_PASSWORD", "urlshortener"),
			DB:       getEnv("POSTGRES_DB", "urlshortener"),
		},
		Redis: RedisConfig{
			Host: getEnv("REDIS_HOST", "localhost"),
			Port: getEnv("REDIS_PORT", "6379"),
		},
		JWT: JWTConfig{
			Secret: getEnv("JWT_SECRET", ""),
		},
		Stripe: StripeConfig{
			SecretKey:     getEnv("STRIPE_SECRET_KEY", ""),
			WebhookSecret: getEnv("STRIPE_WEBHOOK_SECRET", ""),
		},
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}
