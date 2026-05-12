package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

// Config holds all fnclaude configuration, merged from defaults, the config
// file, and environment variables (env overrides config, config overrides
// built-in defaults).
type Config struct {
	Name NameConfig
	Auto AutoConfig
}

// NameConfig holds fields from the [name] TOML section.
// These are loaded and stored but not yet used for behavior.
type NameConfig struct {
	Model            string
	Timeout          time.Duration
	QuietMissingAPIKey bool
}

// AutoConfig holds fields from the [auto] TOML section.
type AutoConfig struct {
	// Tmux controls automatic --tmux injection: "never", "worktree", "always".
	Tmux string

	// IDE controls automatic --ide injection: "never", "always".
	IDE string

	// DangerouslySkipPermissions controls automatic --dangerously-skip-permissions
	// injection.
	DangerouslySkipPermissions bool
}

// rawConfig mirrors the TOML file structure for unmarshalling.
type rawConfig struct {
	Name struct {
		Model            string `toml:"model"`
		Timeout          string `toml:"timeout"`
		QuietMissingAPIKey bool   `toml:"quiet_missing_api_key"`
	} `toml:"name"`
	Auto struct {
		Tmux                       string `toml:"tmux"`
		IDE                        string `toml:"ide"`
		DangerouslySkipPermissions bool   `toml:"dangerously_skip_permissions"`
	} `toml:"auto"`
}

// defaultConfig returns the built-in defaults.
func defaultConfig() Config {
	return Config{
		Name: NameConfig{
			Model:              "claude-haiku-4-5",
			Timeout:            3 * time.Second,
			QuietMissingAPIKey: false,
		},
		Auto: AutoConfig{
			Tmux:                       "never",
			IDE:                        "never",
			DangerouslySkipPermissions: false,
		},
	}
}

// configFilePath returns the path to the config file, honoring XDG_CONFIG_HOME.
func configFilePath() string {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "~"
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "fnclaude", "config.toml")
}

// loadConfig loads the configuration from the config file and environment
// variables, merging over built-in defaults. Order of precedence (high to low):
//
//	env var > config file > built-in default
//
// A missing config file is not an error. A malformed config file prints a
// warning and falls back to defaults.
func loadConfig() Config {
	cfg := defaultConfig()

	// Load config file.
	path := configFilePath()
	if _, err := os.Stat(path); err == nil {
		var raw rawConfig
		if _, err := toml.DecodeFile(path, &raw); err != nil {
			fmt.Fprintf(os.Stderr, "fnclaude: config file %s is malformed, using defaults: %v\n", path, err)
		} else {
			if raw.Name.Model != "" {
				cfg.Name.Model = raw.Name.Model
			}
			if raw.Name.Timeout != "" {
				if d, err := time.ParseDuration(raw.Name.Timeout); err == nil {
					cfg.Name.Timeout = d
				} else {
					fmt.Fprintf(os.Stderr, "fnclaude: invalid timeout %q in config, using default: %v\n", raw.Name.Timeout, err)
				}
			}
			cfg.Name.QuietMissingAPIKey = raw.Name.QuietMissingAPIKey
			if raw.Auto.Tmux != "" {
				cfg.Auto.Tmux = raw.Auto.Tmux
			}
			if raw.Auto.IDE != "" {
				cfg.Auto.IDE = raw.Auto.IDE
			}
			cfg.Auto.DangerouslySkipPermissions = raw.Auto.DangerouslySkipPermissions
		}
	}

	// Apply environment variable overrides. Each field is resolved independently.
	if v := os.Getenv("FNCLAUDE_NAME_MODEL"); v != "" {
		cfg.Name.Model = v
	}
	if v := os.Getenv("FNCLAUDE_NAME_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Name.Timeout = d
		} else {
			fmt.Fprintf(os.Stderr, "fnclaude: invalid FNCLAUDE_NAME_TIMEOUT %q, using current value: %v\n", v, err)
		}
	}
	if v := os.Getenv("FNCLAUDE_QUIET_MISSING_API_KEY"); v != "" {
		cfg.Name.QuietMissingAPIKey = parseBoolEnv(v)
	}
	if v := os.Getenv("FNCLAUDE_TMUX"); v != "" {
		cfg.Auto.Tmux = v
	}
	if v := os.Getenv("FNCLAUDE_IDE"); v != "" {
		cfg.Auto.IDE = v
	}
	if v := os.Getenv("FNCLAUDE_DANGEROUSLY_SKIP_PERMISSIONS"); v != "" {
		cfg.Auto.DangerouslySkipPermissions = parseBoolEnv(v)
	}

	return cfg
}

// parseBoolEnv returns true for "1", "true", "yes" (case-insensitive),
// false for anything else.
func parseBoolEnv(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes":
		return true
	default:
		return false
	}
}
