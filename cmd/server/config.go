package main

import (
	"log/slog"
	"os"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	// Database
	DatabaseURL string
	DBPath      string

	// Server
	Port string

	// Session
	SessionSecret string

	// Admin seed credentials (used when no admin exists yet)
	AdminUsername string
	AdminPassword string

	// Push Notifications
	VAPIDPublicKey  string
	VAPIDPrivateKey string

	// Google OAuth (guardian account login)
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string

	// Supabase Storage (player ID card photo uploads)
	SupabaseURL    string
	SupabaseKey    string
	SupabaseBucket string
}

// LoadConfig reads configuration from environment variables and applies defaults.
func LoadConfig() Config {
	cfg := Config{
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		DBPath:          getEnvOrDefault("DB_PATH", "table_tennis.db"),
		Port:            getEnvOrDefault("PORT", "8080"),
		SessionSecret:   os.Getenv("SESSION_SECRET"),
		AdminUsername:   getEnvOrDefault("ADMIN_USERNAME", "admin"),
		AdminPassword:   getEnvOrDefault("ADMIN_PASSWORD", "password"),
		VAPIDPublicKey:  os.Getenv("VAPID_PUBLIC_KEY"),
		VAPIDPrivateKey: os.Getenv("VAPID_PRIVATE_KEY"),

		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURL:  getEnvOrDefault("GOOGLE_REDIRECT_URL", "http://localhost:8080/account/google/callback"),

		SupabaseURL:    os.Getenv("SUPABASE_URL"),
		SupabaseKey:    os.Getenv("SUPABASE_SERVICE_ROLE_KEY"),
		SupabaseBucket: getEnvOrDefault("SUPABASE_STORAGE_BUCKET", "player-ids"),
	}

	if cfg.SessionSecret == "" {
		slog.Warn("SESSION_SECRET not set — using an insecure default. Set SESSION_SECRET in production!")
		cfg.SessionSecret = "change-me-in-production-32-bytes!"
	}

	return cfg
}

func getEnvOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
