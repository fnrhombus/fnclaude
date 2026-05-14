package main

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	rePathSafeBad = regexp.MustCompile(`[^A-Za-z0-9._/-]+`)
	reDashRun     = regexp.MustCompile(`-{2,}`)
	reSlashRun    = regexp.MustCompile(`/{2,}`)
)

// sanitizeForPath produces a slug safe for both filesystem path components
// and git ref names: collapses anything outside [A-Za-z0-9._/-] to '-',
// dedupes hyphen and slash runs, strips leading [-.] and trailing [-/].
// '/' is allowed so git-style nested refs (feat/foo, team/x/y/z) pass
// through and produce nested worktree paths.
//
// Returns ("", false) when:
//   - the input is empty
//   - the input starts with '/' (would escape the configured path prefix)
//   - the result reduces to empty after sanitization
//   - the result contains a ".." substring (git ref-format rule; also
//     blocks foo/../bar style path escape)
//
// Caller decides whether to reject, fall back, or pass the original
// through with a warning.
func sanitizeForPath(s string) (string, bool) {
	if s == "" {
		return "", false
	}
	if strings.HasPrefix(s, "/") {
		return "", false
	}
	s = rePathSafeBad.ReplaceAllString(s, "-")
	s = reDashRun.ReplaceAllString(s, "-")
	s = reSlashRun.ReplaceAllString(s, "/")
	s = strings.TrimLeft(s, "-.")
	s = strings.TrimRight(s, "-/")
	if s == "" {
		return "", false
	}
	if strings.Contains(s, "..") {
		return "", false
	}
	return s, true
}

// sanitizeNamesInPassthrough scans p for --name/--name=VAL/-n/-n=VAL and
// rewrites VAL to a path-safe form when it contains unsafe chars. Returns
// the modified slice plus one warning message per affected occurrence.
//
// Values that reduce to empty after sanitization are left untouched; we
// only warn. This preserves the user's literal input so claude (or a
// downstream hook) can surface the real error rather than fnclaude
// silently substituting a synthetic name.
func sanitizeNamesInPassthrough(p []string) ([]string, []string) {
	out := make([]string, len(p))
	copy(out, p)
	var warnings []string

	for i := 0; i < len(out); i++ {
		t := out[i]
		switch {
		case (t == "--name" || t == "-n") && i+1 < len(out):
			val := out[i+1]
			cleaned, w, replace := decideSanitize(val)
			if w != "" {
				warnings = append(warnings, w)
			}
			if replace {
				out[i+1] = cleaned
			}
			i++ // skip the value slot
		case strings.HasPrefix(t, "--name="):
			val := t[len("--name="):]
			cleaned, w, replace := decideSanitize(val)
			if w != "" {
				warnings = append(warnings, w)
			}
			if replace {
				out[i] = "--name=" + cleaned
			}
		case strings.HasPrefix(t, "-n="):
			val := t[len("-n="):]
			cleaned, w, replace := decideSanitize(val)
			if w != "" {
				warnings = append(warnings, w)
			}
			if replace {
				out[i] = "-n=" + cleaned
			}
		}
	}
	return out, warnings
}

// decideSanitize returns (cleaned, warning, replace). When the value is
// already clean, returns ("", "", false). When it changed, returns the
// new value, a warning, and true. When it reduced to empty, returns the
// original value, a warning, and false (caller leaves it untouched).
func decideSanitize(val string) (string, string, bool) {
	cleaned, ok := sanitizeForPath(val)
	if !ok {
		return val, fmt.Sprintf(
			"fnclaude: --name %q has no path-safe characters; passing through unchanged",
			val), false
	}
	if cleaned == val {
		return "", "", false
	}
	return cleaned, fmt.Sprintf(
		"fnclaude: --name %q sanitized to %q (illegal path/branch chars)",
		val, cleaned), true
}
