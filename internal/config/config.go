// Package config loads runtime configuration from environment variables.
package config

import "os"

type Config struct {
	GitHubPAT string
	Port      string
}

// MissingPATMessage is shown to the user (server still starts so /healthz
// works and the UI can render this as a friendly page instead of the
// process crash-looping) when GITHUB_PAT isn't configured.
const MissingPATMessage = "GITHUB_PAT is not set; create a classic Personal Access Token with 'read:org' and 'repo' scopes at https://github.com/settings/tokens and put it in .env, then restart."

func Load() Config {
	cfg := Config{
		GitHubPAT: os.Getenv("GITHUB_PAT"),
		Port:      os.Getenv("PORT"),
	}
	if cfg.Port == "" {
		cfg.Port = "8080"
	}
	return cfg
}
