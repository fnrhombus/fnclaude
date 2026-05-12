package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// shouldAutoName returns true when the passthrough slice meets all conditions
// for automatic --name injection:
//
//   - contains "--" followed by at least one non-empty token
//   - does NOT already contain --name / -n / --name=* / -n=*
//   - does NOT contain -p / --print
//   - does NOT contain -r / --resume / -r=* / --resume=*
//   - does NOT contain -c / --continue
//   - does NOT contain --from-pr / --from-pr=* / -P / -P=*
func shouldAutoName(passthrough []string) bool {
	// Find "--" and verify at least one non-empty token follows.
	sepIdx := -1
	for i, t := range passthrough {
		if t == "--" {
			sepIdx = i
			break
		}
	}
	if sepIdx < 0 {
		return false
	}
	hasPrompt := false
	for _, t := range passthrough[sepIdx+1:] {
		if t != "" {
			hasPrompt = true
			break
		}
	}
	if !hasPrompt {
		return false
	}

	// Check for disqualifying tokens.
	for _, t := range passthrough {
		switch {
		case t == "--name", t == "-n",
			strings.HasPrefix(t, "--name="), strings.HasPrefix(t, "-n="):
			return false
		case t == "-p", t == "--print":
			return false
		case t == "-r", t == "--resume",
			strings.HasPrefix(t, "-r="), strings.HasPrefix(t, "--resume="):
			return false
		case t == "-c", t == "--continue":
			return false
		case t == "--from-pr", strings.HasPrefix(t, "--from-pr="),
			t == "-P", strings.HasPrefix(t, "-P="):
			return false
		}
	}
	return true
}

// extractPrompt returns the first non-empty token after "--" in passthrough.
// Returns "" if not found.
func extractPrompt(passthrough []string) string {
	sepIdx := -1
	for i, t := range passthrough {
		if t == "--" {
			sepIdx = i
			break
		}
	}
	if sepIdx < 0 {
		return ""
	}
	for _, t := range passthrough[sepIdx+1:] {
		if t != "" {
			return t
		}
	}
	return ""
}

// stopWords is the set of words dropped by heuristicName.
var stopWords = map[string]bool{
	"a": true, "an": true, "the": true,
	"is": true, "are": true, "was": true, "were": true,
	"do": true, "does": true, "did": true,
	"of": true, "for": true, "to": true, "in": true,
	"on": true, "at": true, "with": true,
	"this": true, "that": true,
	"please": true, "can": true, "could": true, "would": true, "should": true,
}

// heuristicName derives a session name from a prompt without any LLM call.
func heuristicName(prompt string) string {
	lower := strings.ToLower(prompt)
	words := strings.Fields(lower)

	var kept []string
	for _, w := range words {
		if stopWords[w] {
			continue
		}
		// Strip non-alphanumeric characters.
		var b strings.Builder
		for _, r := range w {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				b.WriteRune(r)
			}
		}
		clean := b.String()
		if clean != "" {
			kept = append(kept, clean)
		}
		if len(kept) == 3 {
			break
		}
	}

	if len(kept) == 0 {
		return "session"
	}
	return strings.Join(kept, "-")
}

var (
	reNonSlug    = regexp.MustCompile(`[^a-z0-9-]+`)
	reMultiDash  = regexp.MustCompile(`-{2,}`)
	reWhitespace = regexp.MustCompile(`\s+`)
)

// sanitizeName cleans raw LLM output into a valid slug.
func sanitizeName(raw string) string {
	s := strings.TrimSpace(raw)
	s = strings.ToLower(s)
	// Replace whitespace runs with "-".
	s = reWhitespace.ReplaceAllString(s, "-")
	// Strip anything not in [a-z0-9-].
	s = reNonSlug.ReplaceAllString(s, "")
	// Collapse consecutive dashes.
	s = reMultiDash.ReplaceAllString(s, "-")
	// Trim leading/trailing dashes.
	s = strings.Trim(s, "-")
	// Take first 3 dash-segments.
	parts := strings.SplitN(s, "-", 4)
	if len(parts) > 3 {
		parts = parts[:3]
	}
	s = strings.Join(parts, "-")
	// Trim again in case joining re-introduced edge dashes.
	s = strings.Trim(s, "-")
	return s
}

const nameSystemPrompt = "Generate a 1-3 word lowercase hyphen-separated label for this user's request. Output ONLY the label — no punctuation, no quotes, no explanation, no leading 'Label:'. Examples: 'fix-login-bug', 'add-dark-mode', 'refactor-auth'."

// llmClientFunc is the signature for the LLM call, injectable for testing.
type llmClientFunc func(ctx context.Context, model, prompt string) (string, error)

// defaultLLMClient calls the real Anthropic API.
func defaultLLMClient(apiKey string) llmClientFunc {
	return func(ctx context.Context, model, prompt string) (string, error) {
		client := anthropic.NewClient(option.WithAPIKey(apiKey))
		msg, err := client.Messages.New(ctx, anthropic.MessageNewParams{
			Model:     anthropic.Model(model),
			MaxTokens: 30,
			System: []anthropic.TextBlockParam{
				{Text: nameSystemPrompt},
			},
			Messages: []anthropic.MessageParam{
				anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
			},
		})
		if err != nil {
			return "", err
		}
		for _, blk := range msg.Content {
			if blk.Type == "text" {
				return blk.Text, nil
			}
		}
		return "", fmt.Errorf("no text block in response")
	}
}

// generateName produces a session name for the given prompt.
// stderr is used to emit the missing-API-key warning.
// llmFn may be nil to force the heuristic path (e.g. in tests).
func generateName(prompt string, cfg NameConfig, apiKey string, llmFn llmClientFunc, stderr *os.File) string {
	// API key missing: warn (unless suppressed) then fall back.
	if apiKey == "" {
		if !cfg.QuietMissingAPIKey {
			fmt.Fprintln(stderr, "fnclaude: ANTHROPIC_API_KEY not set; using heuristic name (suppress with FNCLAUDE_QUIET_MISSING_API_KEY=1 or config quiet_missing_api_key = true)")
		}
		return heuristicName(prompt)
	}

	if llmFn == nil {
		llmFn = defaultLLMClient(apiKey)
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	raw, err := llmFn(ctx, cfg.Model, prompt)
	if err != nil {
		// Silent fall-through on any error.
		return heuristicName(prompt)
	}

	name := sanitizeName(raw)
	if name == "" {
		return heuristicName(prompt)
	}
	return name
}
