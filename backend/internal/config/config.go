package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Server   ServerConfig
	Postgres PostgresConfig
	Redis    RedisConfig
	JWT      JWTConfig
	Stripe   StripeConfig
	Mailer   MailerConfig
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

type MailerConfig struct {
	Transport    string // "console" | "smtp" | "sendmail"
	From         string // sender address
	FromName     string // display name in From header
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	SMTPTLS      bool   // true = STARTTLS (port 587); false = plain (port 25)
	SendmailPath string // absolute path to sendmail binary
}

func Load() (*Config, error) {
	_ = godotenv.Load("../../.env")

	cfg := &Config{
		Server: ServerConfig{
			Port:            getEnv("SERVER_PORT", "8080"),
			Mode:            getEnv("GIN_MODE", "debug"),
			BaseURL:         getEnv("BASE_URL", "http://localhost:8080"),
			FrontendBaseURL: getEnv("FRONTEND_BASE_URL", "http://localhost:3000"),
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
		Mailer: MailerConfig{
			Transport:    getEnv("MAIL_TRANSPORT", "console"),
			From:         getEnv("MAIL_FROM", "noreply@localhost"),
			FromName:     getEnv("MAIL_FROM_NAME", "URL Shortener"),
			SMTPHost:     getEnv("SMTP_HOST", ""),
			SMTPPort:     getEnvInt("SMTP_PORT", 587),
			SMTPUsername: getEnv("SMTP_USERNAME", ""),
			SMTPPassword: getEnv("SMTP_PASSWORD", ""),
			SMTPTLS:      getEnvBool("SMTP_TLS", true),
			SendmailPath: getEnv("SENDMAIL_PATH", "/usr/sbin/sendmail"),
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

func getEnvInt(key string, fallback int) int {
	if val, ok := os.LookupEnv(key); ok {
		if n, err := strconv.Atoi(val); err == nil {
			return n
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if val, ok := os.LookupEnv(key); ok {
		if b, err := strconv.ParseBool(val); err == nil {
			return b
		}
	}
	return fallback
}
