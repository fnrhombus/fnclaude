package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ── defaultConfig tests ────────────────────────────────────────────────────

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()
	if cfg.Name.Model != "claude-haiku-4-5" {
		t.Errorf("Name.Model: got %q, want %q", cfg.Name.Model, "claude-haiku-4-5")
	}
	if cfg.Name.Timeout != 3*time.Second {
		t.Errorf("Name.Timeout: got %v, want 3s", cfg.Name.Timeout)
	}
	if cfg.Name.QuietMissingAPIKey {
		t.Error("Name.QuietMissingAPIKey: got true, want false")
	}
	if cfg.Auto.Tmux != "never" {
		t.Errorf("Auto.Tmux: got %q, want %q", cfg.Auto.Tmux, "never")
	}
	if cfg.Auto.IDE != "never" {
		t.Errorf("Auto.IDE: got %q, want %q", cfg.Auto.IDE, "never")
	}
	if cfg.Auto.DangerouslySkipPermissions {
		t.Error("Auto.DangerouslySkipPermissions: got true, want false")
	}
}

// ── parseBoolEnv tests ─────────────────────────────────────────────────────

func TestParseBoolEnv(t *testing.T) {
	trueVals := []string{"1", "true", "True", "TRUE", "yes", "YES", "Yes"}
	for _, v := range trueVals {
		if !parseBoolEnv(v) {
			t.Errorf("parseBoolEnv(%q): got false, want true", v)
		}
	}
	falseVals := []string{"0", "false", "no", "maybe", ""}
	for _, v := range falseVals {
		if parseBoolEnv(v) {
			t.Errorf("parseBoolEnv(%q): got true, want false", v)
		}
	}
}

// ── loadConfig tests ───────────────────────────────────────────────────────

// writeConfigFile writes content to a temp dir and sets XDG_CONFIG_HOME so
// loadConfig picks it up. Returns a cleanup func.
func writeConfigFile(t *testing.T, content string) func() {
	t.Helper()
	dir := t.TempDir()
	cfgDir := filepath.Join(dir, "fnclaude")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", dir)
	return func() {} // t.TempDir and t.Setenv clean up automatically
}

func TestLoadConfig_NoFile_UsesDefaults(t *testing.T) {
	// Point XDG_CONFIG_HOME to an empty temp dir (no config.toml).
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	// Clear env vars that might leak from outer environment.
	clearConfigEnv(t)

	cfg := loadConfig()
	def := defaultConfig()
	if cfg.Name.Model != def.Name.Model {
		t.Errorf("Name.Model: got %q, want %q", cfg.Name.Model, def.Name.Model)
	}
	if cfg.Auto.Tmux != def.Auto.Tmux {
		t.Errorf("Auto.Tmux: got %q, want %q", cfg.Auto.Tmux, def.Auto.Tmux)
	}
}

func TestLoadConfig_FileOverridesDefaults(t *testing.T) {
	writeConfigFile(t, `
[name]
model = "claude-opus-4-5"
timeout = "10s"
quiet_missing_api_key = true

[auto]
tmux = "worktree"
ide = "always"
dangerously_skip_permissions = true
`)
	clearConfigEnv(t)

	cfg := loadConfig()
	if cfg.Name.Model != "claude-opus-4-5" {
		t.Errorf("Name.Model: got %q", cfg.Name.Model)
	}
	if cfg.Name.Timeout != 10*time.Second {
		t.Errorf("Name.Timeout: got %v", cfg.Name.Timeout)
	}
	if !cfg.Name.QuietMissingAPIKey {
		t.Error("Name.QuietMissingAPIKey: got false, want true")
	}
	if cfg.Auto.Tmux != "worktree" {
		t.Errorf("Auto.Tmux: got %q", cfg.Auto.Tmux)
	}
	if cfg.Auto.IDE != "always" {
		t.Errorf("Auto.IDE: got %q", cfg.Auto.IDE)
	}
	if !cfg.Auto.DangerouslySkipPermissions {
		t.Error("Auto.DangerouslySkipPermissions: got false, want true")
	}
}

func TestLoadConfig_MalformedFile_FallsBackToDefaults(t *testing.T) {
	writeConfigFile(t, `this is not valid toml ][[[`)
	clearConfigEnv(t)

	cfg := loadConfig()
	def := defaultConfig()
	// Should fall back to defaults on malformed file.
	if cfg.Name.Model != def.Name.Model {
		t.Errorf("Name.Model: got %q, want default %q", cfg.Name.Model, def.Name.Model)
	}
}

func TestLoadConfig_EnvOverridesFile(t *testing.T) {
	writeConfigFile(t, `
[name]
model = "claude-haiku-4-5"

[auto]
tmux = "worktree"
dangerously_skip_permissions = false
`)
	clearConfigEnv(t)
	t.Setenv("FNCLAUDE_NAME_MODEL", "claude-sonnet-4-5")
	t.Setenv("FNCLAUDE_TMUX", "never")
	t.Setenv("FNCLAUDE_DANGEROUSLY_SKIP_PERMISSIONS", "true")

	cfg := loadConfig()
	if cfg.Name.Model != "claude-sonnet-4-5" {
		t.Errorf("Name.Model: got %q, want claude-sonnet-4-5", cfg.Name.Model)
	}
	// File set tmux="worktree"; env overrides to "never".
	if cfg.Auto.Tmux != "never" {
		t.Errorf("Auto.Tmux: got %q, want never (env-override)", cfg.Auto.Tmux)
	}
	if !cfg.Auto.DangerouslySkipPermissions {
		t.Error("Auto.DangerouslySkipPermissions: got false, want true")
	}
}

func TestLoadConfig_EnvTimeout(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	clearConfigEnv(t)
	t.Setenv("FNCLAUDE_NAME_TIMEOUT", "15s")

	cfg := loadConfig()
	if cfg.Name.Timeout != 15*time.Second {
		t.Errorf("Name.Timeout: got %v, want 15s", cfg.Name.Timeout)
	}
}

func TestLoadConfig_PartialFile_UnsetFieldsStayDefault(t *testing.T) {
	// Only set one field; others should remain at built-in defaults.
	writeConfigFile(t, `
[auto]
tmux = "worktree"
`)
	clearConfigEnv(t)

	cfg := loadConfig()
	if cfg.Auto.Tmux != "worktree" {
		t.Errorf("Auto.Tmux: got %q, want always", cfg.Auto.Tmux)
	}
	// IDE not in file — should still be default.
	if cfg.Auto.IDE != "never" {
		t.Errorf("Auto.IDE: got %q, want never", cfg.Auto.IDE)
	}
	// Model not in file — should still be default.
	if cfg.Name.Model != "claude-haiku-4-5" {
		t.Errorf("Name.Model: got %q, want claude-haiku-4-5", cfg.Name.Model)
	}
}

// clearConfigEnv unsets all FNCLAUDE_* env vars so tests don't bleed into
// each other when the test runner inherits them from the environment.
func clearConfigEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"FNCLAUDE_NAME_MODEL",
		"FNCLAUDE_NAME_TIMEOUT",
		"FNCLAUDE_QUIET_MISSING_API_KEY",
		"FNCLAUDE_TMUX",
		"FNCLAUDE_IDE",
		"FNCLAUDE_DANGEROUSLY_SKIP_PERMISSIONS",
	} {
		t.Setenv(k, "")
		os.Unsetenv(k)
	}
}

// ── configFilePath tests ──────────────────────────────────────────────────

func TestConfigFilePath_XDGSet_UsesXDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/custom/xdg")
	got := configFilePath()
	want := "/custom/xdg/fnclaude/config.toml"
	if got != want {
		t.Errorf("configFilePath() = %q, want %q", got, want)
	}
}

func TestConfigFilePath_XDGUnset_UsesHomeDotConfig(t *testing.T) {
	// Unset XDG_CONFIG_HOME so the fallback ($HOME/.config) branch fires.
	os.Unsetenv("XDG_CONFIG_HOME")
	t.Setenv("HOME", "/fake/home")
	got := configFilePath()
	want := "/fake/home/.config/fnclaude/config.toml"
	if got != want {
		t.Errorf("configFilePath() = %q, want %q", got, want)
	}
}

// ── loadConfig invalid-value branches ─────────────────────────────────────

func TestLoadConfig_InvalidTimeoutInFile_WarnsAndKeepsDefault(t *testing.T) {
	writeConfigFile(t, `
[name]
timeout = "not-a-duration"
`)
	clearConfigEnv(t)
	deferredWarnings = nil
	t.Cleanup(func() { deferredWarnings = nil })

	cfg := loadConfig()
	def := defaultConfig()
	if cfg.Name.Timeout != def.Name.Timeout {
		t.Errorf("Name.Timeout: got %v, want default %v", cfg.Name.Timeout, def.Name.Timeout)
	}
	found := false
	for _, w := range deferredWarnings {
		if strings.Contains(w, "invalid timeout") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected deferred warning about invalid timeout, got: %v", deferredWarnings)
	}
}

func TestLoadConfig_InvalidEnvTimeout_WarnsAndKeepsCurrent(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	clearConfigEnv(t)
	t.Setenv("FNCLAUDE_NAME_TIMEOUT", "garbage")
	deferredWarnings = nil
	t.Cleanup(func() { deferredWarnings = nil })

	cfg := loadConfig()
	def := defaultConfig()
	if cfg.Name.Timeout != def.Name.Timeout {
		t.Errorf("Name.Timeout: got %v, want default %v", cfg.Name.Timeout, def.Name.Timeout)
	}
	found := false
	for _, w := range deferredWarnings {
		if strings.Contains(w, "FNCLAUDE_NAME_TIMEOUT") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected deferred warning about FNCLAUDE_NAME_TIMEOUT, got: %v", deferredWarnings)
	}
}

func TestLoadConfig_EnvQuietMissingAPIKey(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	clearConfigEnv(t)
	t.Setenv("FNCLAUDE_QUIET_MISSING_API_KEY", "1")

	cfg := loadConfig()
	if !cfg.Name.QuietMissingAPIKey {
		t.Error("Name.QuietMissingAPIKey: got false, want true")
	}
}

func TestLoadConfig_EnvIDE(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	clearConfigEnv(t)
	t.Setenv("FNCLAUDE_IDE", "always")

	cfg := loadConfig()
	if cfg.Auto.IDE != "always" {
		t.Errorf("Auto.IDE: got %q, want always", cfg.Auto.IDE)
	}
}

// ── normalizeTmuxMode tests ───────────────────────────────────────────────

func TestNormalizeTmuxMode(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"never", "never"},
		{"worktree", "worktree"},
		{"", "never"},        // empty → never (no warning)
		{"always", "never"},  // deprecated → never (warning, covered elsewhere)
		{"garbage", "never"}, // unknown → never (warning)
	}
	for _, tc := range cases {
		deferredWarnings = nil
		got := normalizeTmuxMode(tc.in)
		if got != tc.want {
			t.Errorf("normalizeTmuxMode(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
	deferredWarnings = nil
}

func TestLoadConfig_InvalidTmuxValue_NormalizedToNeverWithWarning(t *testing.T) {
	// Deprecated "always" value (or any unknown string) should normalize to
	// "never" and queue a stderr warning that fnclaude will flush after exit.
	writeConfigFile(t, `
[auto]
tmux = "always"
`)
	clearConfigEnv(t)
	deferredWarnings = nil
	t.Cleanup(func() { deferredWarnings = nil })

	cfg := loadConfig()
	if cfg.Auto.Tmux != "never" {
		t.Errorf("Auto.Tmux: got %q, want never (fallback for invalid value)", cfg.Auto.Tmux)
	}
	found := false
	for _, w := range deferredWarnings {
		if strings.Contains(w, "auto.tmux=\"always\"") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected a deferred warning about auto.tmux, got: %v", deferredWarnings)
	}
}
