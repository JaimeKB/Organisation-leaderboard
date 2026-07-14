// Package config loads runtime configuration (env vars, label config file).
package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// AppConfig holds settings read from the environment.
type AppConfig struct {
	GitHubPAT string
	Port      string
}

// Load reads a .env file if present (ignored if missing, e.g. inside Docker
// where --env-file already populated the process environment) and returns
// the resolved AppConfig, failing fast if required values are absent.
func Load() (AppConfig, error) {
	_ = godotenv.Load()

	pat := os.Getenv("GITHUB_PAT")
	if pat == "" {
		return AppConfig{}, fmt.Errorf("GITHUB_PAT environment variable is required")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	return AppConfig{GitHubPAT: pat, Port: port}, nil
}
