package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port       string
	DBPath     string
	JWTSecret  string
	AdminUser  string
	AdminPass  string
	SMTPHost   string
	SMTPPort   string
	SMTPUser   string
	SMTPPass   string
	SMTPFrom   string
	FrontendURL string
}

var App Config

func Load() {
	_ = godotenv.Load()

	App = Config{
		Port:        getEnv("PORT", "8080"),
		DBPath:      getEnv("DB_PATH", "./digistore.db"),
		JWTSecret:   getEnv("JWT_SECRET", "ganti-secret-ini-di-production"),
		AdminUser:   getEnv("ADMIN_USERNAME", "admin"),
		AdminPass:   getEnv("ADMIN_PASSWORD", "admin123"),
		SMTPHost:    getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:    getEnv("SMTP_PORT", "587"),
		SMTPUser:    getEnv("SMTP_USER", ""),
		SMTPPass:    getEnv("SMTP_PASS", ""),
		SMTPFrom:    getEnv("SMTP_FROM", "noreply@digistore.id"),
		FrontendURL: getEnv("FRONTEND_URL", "http://localhost:5173"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
