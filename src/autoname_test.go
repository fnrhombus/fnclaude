package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// ── shouldAutoName ─────────────────────────────────────────────────────────

func TestShouldAutoName(t *testing.T) {
	cases := []struct {
		name        string
		passthrough []string
		want        bool
	}{
		// ── should fire ───────────────────────────────────────────────────────
		{
			name:        "bare dash-dash with prompt",
			passthrough: []string{"--", "fix the login bug"},
			want:        true,
		},
		{
			name:        "flags before dash-dash then prompt",
			passthrough: []string{"--verbose", "--", "add dark mode"},
			want:        true,
		},
		{
			name:        "multiple tokens after dash-dash — only first used but still fires",
			passthrough: []string{"--", "refactor auth", "extra"},
			want:        true,
		},

		// ── should NOT fire: no dash-dash ─────────────────────────────────────
		{
			name:        "no dash-dash",
			passthrough: []string{"--verbose", "something"},
			want:        false,
		},
		{
			name:        "empty",
			passthrough: []string{},
			want:        false,
		},

		// ── should NOT fire: dash-dash but no prompt after ─────────────────────
		{
			name:        "dash-dash at end, no prompt",
			passthrough: []string{"--"},
			want:        false,
		},
		{
			name:        "dash-dash followed only by empty strings",
			passthrough: []string{"--", "", ""},
			want:        false,
		},

		// ── should NOT fire: --name already present ────────────────────────────
		{
			name:        "--name bare",
			passthrough: []string{"--name", "foo", "--", "prompt"},
			want:        false,
		},
		{
			name:        "--name=val",
			passthrough: []string{"--name=mysession", "--", "prompt"},
			want:        false,
		},
		{
			name:        "-n bare",
			passthrough: []string{"-n", "foo", "--", "prompt"},
			want:        false,
		},
		{
			name:        "-n=val",
			passthrough: []string{"-n=foo", "--", "prompt"},
			want:        false,
		},

		// ── should NOT fire: --print ───────────────────────────────────────────
		{
			name:        "-p",
			passthrough: []string{"-p", "--", "prompt"},
			want:        false,
		},
		{
			name:        "--print",
			passthrough: []string{"--print", "--", "prompt"},
			want:        false,
		},

		// ── should NOT fire: --resume ──────────────────────────────────────────
		{
			name:        "-r bare",
			passthrough: []string{"-r", "--", "prompt"},
			want:        false,
		},
		{
			name:        "--resume bare",
			passthrough: []string{"--resume", "--", "prompt"},
			want:        false,
		},
		{
			name:        "-r=val",
			passthrough: []string{"-r=abc123", "--", "prompt"},
			want:        false,
		},
		{
			name:        "--resume=val",
			passthrough: []string{"--resume=abc123", "--", "prompt"},
			want:        false,
		},

		// ── should NOT fire: --continue ────────────────────────────────────────
		{
			name:        "-c",
			passthrough: []string{"-c", "--", "prompt"},
			want:        false,
		},
		{
			name:        "--continue",
			passthrough: []string{"--continue", "--", "prompt"},
			want:        false,
		},

		// ── should NOT fire: --from-pr / -P ───────────────────────────────────
		{
			name:        "--from-pr bare",
			passthrough: []string{"--from-pr", "--", "prompt"},
			want:        false,
		},
		{
			name:        "--from-pr=val",
			passthrough: []string{"--from-pr=123", "--", "prompt"},
			want:        false,
		},
		{
			name:        "-P bare",
			passthrough: []string{"-P", "--", "prompt"},
			want:        false,
		},
		{
			name:        "-P=val",
			passthrough: []string{"-P=123", "--", "prompt"},
			want:        false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldAutoName(tc.passthrough)
			if got != tc.want {
				t.Errorf("shouldAutoName(%v) = %v, want %v", tc.passthrough, got, tc.want)
			}
		})
	}
}

// ── extractPrompt ─────────────────────────────────────────────────────────

func TestExtractPrompt(t *testing.T) {
	cases := []struct {
		name        string
		passthrough []string
		want        string
	}{
		{
			name:        "first non-empty token after --",
			passthrough: []string{"--verbose", "--", "fix login bug", "extra"},
			want:        "fix login bug",
		},
		{
			name:        "no dash-dash present",
			passthrough: []string{"--verbose", "something"},
			want:        "",
		},
		{
			name:        "dash-dash at end, no tokens after",
			passthrough: []string{"--verbose", "--"},
			want:        "",
		},
		{
			name:        "dash-dash followed only by empty strings",
			passthrough: []string{"--", "", "", ""},
			want:        "",
		},
		{
			name:        "empty tokens skipped, first non-empty wins",
			passthrough: []string{"--", "", "real prompt"},
			want:        "real prompt",
		},
		{
			name:        "empty passthrough",
			passthrough: []string{},
			want:        "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractPrompt(tc.passthrough)
			if got != tc.want {
				t.Errorf("extractPrompt(%v) = %q, want %q", tc.passthrough, got, tc.want)
			}
		})
	}
}

// ── heuristicName ─────────────────────────────────────────────────────────

func TestHeuristicName(t *testing.T) {
	cases := []struct {
		name   string
		prompt string
		want   string
	}{
		{
			name:   "simple three words",
			prompt: "fix login bug",
			want:   "fix-login-bug",
		},
		{
			name:   "stops stripped",
			prompt: "please fix the login bug now",
			want:   "fix-login-bug",
		},
		{
			name:   "exactly three content words",
			prompt: "add dark mode",
			want:   "add-dark-mode",
		},
		{
			name:   "more than three content words — take first three",
			prompt: "refactor the authentication module completely",
			want:   "refactor-authentication-module",
		},
		{
			name:   "punctuation stripped per word",
			prompt: "fix! login. bug?",
			want:   "fix-login-bug",
		},
		{
			name:   "mixed case lowercased",
			prompt: "Add Dark Mode",
			want:   "add-dark-mode",
		},
		{
			name:   "all stop words",
			prompt: "please do the thing for a",
			want:   "thing",
		},
		{
			name:   "empty prompt",
			prompt: "",
			want:   "session",
		},
		{
			name:   "punctuation only",
			prompt: "!!! ???",
			want:   "session",
		},
		{
			name:   "single word",
			prompt: "refactor",
			want:   "refactor",
		},
		{
			name:   "two words",
			prompt: "fix bug",
			want:   "fix-bug",
		},
		{
			name:   "stop words with no content",
			prompt: "the a an",
			want:   "session",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := heuristicName(tc.prompt)
			if got != tc.want {
				t.Errorf("heuristicName(%q) = %q, want %q", tc.prompt, got, tc.want)
			}
		})
	}
}

// ── sanitizeName ──────────────────────────────────────────────────────────

func TestSanitizeName(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "already clean",
			raw:  "fix-login-bug",
			want: "fix-login-bug",
		},
		{
			name: "mixed case",
			raw:  "Fix-Login-Bug",
			want: "fix-login-bug",
		},
		{
			name: "leading Label: prefix — colon and space become dash, label is a segment",
			raw:  "Label: fix-login-bug",
			want: "label-fix-login",
		},
		{
			name: "trailing dot",
			raw:  "fix-login-bug.",
			want: "fix-login-bug",
		},
		{
			name: "emoji stripped",
			raw:  "🔧 fix-login-bug 🐛",
			want: "fix-login-bug",
		},
		{
			name: "spaces become dashes",
			raw:  "fix login bug",
			want: "fix-login-bug",
		},
		{
			name: "more than three segments truncated",
			raw:  "fix-the-login-bug",
			want: "fix-the-login",
		},
		{
			name: "consecutive dashes collapsed",
			raw:  "fix--login---bug",
			want: "fix-login-bug",
		},
		{
			name: "leading and trailing dashes trimmed",
			raw:  "-fix-login-bug-",
			want: "fix-login-bug",
		},
		{
			name: "quoted output",
			raw:  `"fix-login-bug"`,
			want: "fix-login-bug",
		},
		{
			name: "empty input",
			raw:  "",
			want: "",
		},
		{
			name: "whitespace only",
			raw:  "   ",
			want: "",
		},
		{
			name: "only special chars",
			raw:  "!!??##",
			want: "",
		},
		{
			name: "mixed whitespace and special",
			raw:  "  Add Dark Mode  ",
			want: "add-dark-mode",
		},
		{
			name: "explanation text after label",
			raw:  "add-dark-mode (adds a dark mode toggle)",
			want: "add-dark-mode",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := sanitizeName(tc.raw)
			if got != tc.want {
				t.Errorf("sanitizeName(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}

// ── generateName ──────────────────────────────────────────────────────────

// resetWarnings clears the deferred-warnings buffer at the start of a test
// and restores it (empty) at cleanup. Tests inspect deferredWarnings directly
// after calling generateName.
func resetWarnings(t *testing.T) {
	t.Helper()
	deferredWarnings = nil
	t.Cleanup(func() { deferredWarnings = nil })
}

// warningsContain reports whether any queued warning contains the substring.
func warningsContain(needle string) bool {
	for _, w := range deferredWarnings {
		if strings.Contains(w, needle) {
			return true
		}
	}
	return false
}

func TestGenerateName_HappyPath(t *testing.T) {
	resetWarnings(t)
	cfg := defaultConfig().Name
	apiKey := "test-key"

	llmFn := func(_ context.Context, model, prompt string) (string, error) {
		return "fix-login-bug", nil
	}

	got := generateName("fix the login bug", cfg, apiKey, llmFn)
	if got != "fix-login-bug" {
		t.Errorf("got %q, want %q", got, "fix-login-bug")
	}
	if len(deferredWarnings) != 0 {
		t.Errorf("unexpected warnings: %v", deferredWarnings)
	}
}

func TestGenerateName_LLMOutputSanitized(t *testing.T) {
	resetWarnings(t)
	cfg := defaultConfig().Name
	apiKey := "test-key"

	llmFn := func(_ context.Context, model, prompt string) (string, error) {
		return "Fix Login Bug!!!", nil
	}

	got := generateName("fix the login bug", cfg, apiKey, llmFn)
	if got != "fix-login-bug" {
		t.Errorf("got %q, want %q", got, "fix-login-bug")
	}
}

func TestGenerateName_Timeout_FallsBackToHeuristic(t *testing.T) {
	resetWarnings(t)
	cfg := defaultConfig().Name
	cfg.Timeout = 10 * time.Millisecond
	apiKey := "test-key"

	llmFn := func(ctx context.Context, model, prompt string) (string, error) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(500 * time.Millisecond):
			return "should-not-reach", nil
		}
	}

	got := generateName("fix the login bug", cfg, apiKey, llmFn)
	if got != "fix-login-bug" {
		t.Errorf("got %q, want %q", got, "fix-login-bug")
	}
	if len(deferredWarnings) != 0 {
		t.Errorf("unexpected warnings on timeout (API errors should be silent): %v", deferredWarnings)
	}
}

func TestGenerateName_APIError_FallsBackToHeuristic(t *testing.T) {
	resetWarnings(t)
	cfg := defaultConfig().Name
	apiKey := "test-key"

	llmFn := func(_ context.Context, model, prompt string) (string, error) {
		return "", errors.New("401 Unauthorized")
	}

	got := generateName("add dark mode", cfg, apiKey, llmFn)
	if got != "add-dark-mode" {
		t.Errorf("got %q, want %q", got, "add-dark-mode")
	}
	if len(deferredWarnings) != 0 {
		t.Errorf("API errors should be silent, got: %v", deferredWarnings)
	}
}

func TestGenerateName_EmptyLLMOutput_FallsBackToHeuristic(t *testing.T) {
	resetWarnings(t)
	cfg := defaultConfig().Name
	apiKey := "test-key"

	llmFn := func(_ context.Context, model, prompt string) (string, error) {
		return "!!!", nil // sanitizes to ""
	}

	got := generateName("refactor auth", cfg, apiKey, llmFn)
	if got != "refactor-auth" {
		t.Errorf("got %q, want %q", got, "refactor-auth")
	}
}

func TestGenerateName_MissingAPIKey_WarnsAndFallsBack(t *testing.T) {
	resetWarnings(t)
	cfg := defaultConfig().Name
	apiKey := ""

	got := generateName("add dark mode", cfg, apiKey, nil)
	if got != "add-dark-mode" {
		t.Errorf("got %q, want %q", got, "add-dark-mode")
	}
	if !warningsContain("ANTHROPIC_API_KEY not set") {
		t.Errorf("expected API-key warning to be deferred, got: %v", deferredWarnings)
	}
}

func TestGenerateName_ZeroTimeout_FallsBackToDefault(t *testing.T) {
	// cfg.Timeout = 0 should trigger the "<=0 → 3s default" branch.
	resetWarnings(t)
	cfg := defaultConfig().Name
	cfg.Timeout = 0
	apiKey := "test-key"

	called := false
	llmFn := func(_ context.Context, model, prompt string) (string, error) {
		called = true
		return "fix-login-bug", nil
	}

	got := generateName("fix the login bug", cfg, apiKey, llmFn)
	if !called {
		t.Error("llmFn was not invoked")
	}
	if got != "fix-login-bug" {
		t.Errorf("got %q, want %q", got, "fix-login-bug")
	}
}

func TestGenerateName_MissingAPIKey_QuietMode_NoWarning(t *testing.T) {
	resetWarnings(t)
	cfg := defaultConfig().Name
	cfg.QuietMissingAPIKey = true
	apiKey := ""

	got := generateName("add dark mode", cfg, apiKey, nil)
	if got != "add-dark-mode" {
		t.Errorf("got %q, want %q", got, "add-dark-mode")
	}
	if len(deferredWarnings) != 0 {
		t.Errorf("expected no warnings in quiet mode, got: %v", deferredWarnings)
	}
}
