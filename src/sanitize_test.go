package main

import (
	"reflect"
	"testing"
)

func TestSanitizeForPath(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    string
		wantOk  bool
	}{
		// ── passthrough ────────────────────────────────────────────────────
		{"already safe lowercase", "hello-world", "hello-world", true},
		{"mixed case allowed", "Foo_Bar", "Foo_Bar", true},
		{"versioned", "v1.2.3", "v1.2.3", true},
		{"digits", "abc123", "abc123", true},

		// ── single forbidden chars become hyphens ─────────────────────────
		{"space", "foo bar", "foo-bar", true},
		{"slash", "foo/bar", "foo-bar", true},
		{"backslash", "foo\\bar", "foo-bar", true},
		{"colon", "foo:bar", "foo-bar", true},
		{"star", "foo*bar", "foo-bar", true},
		{"qmark", "foo?bar", "foo-bar", true},
		{"pipe", "foo|bar", "foo-bar", true},
		{"tilde", "foo~bar", "foo-bar", true},
		{"caret", "foo^bar", "foo-bar", true},

		// ── runs collapse ──────────────────────────────────────────────────
		{"multi-space", "foo   bar", "foo-bar", true},
		{"mixed punct", "foo!@#$bar", "foo-bar", true},
		{"double slash", "foo//bar", "foo-bar", true},
		{"run of hyphens", "foo---bar", "foo-bar", true},

		// ── trim leading/trailing ──────────────────────────────────────────
		{"leading hyphen", "-foo", "foo", true},
		{"leading hyphens", "---foo", "foo", true},
		{"leading dot", ".hidden", "hidden", true},
		{"leading dots", "..parent", "parent", true},
		{"leading mixed", ".-.-foo", "foo", true},
		{"trailing hyphen", "foo-", "foo", true},
		{"trailing slash", "foo/", "foo", true},

		// ── middle dots preserved ─────────────────────────────────────────
		{"middle dots", "foo.bar.baz", "foo.bar.baz", true},

		// ── non-ASCII forbidden ────────────────────────────────────────────
		{"accent stripped", "café", "caf", true},
		{"diaeresis", "naïve-attempt", "na-ve-attempt", true},

		// ── empty results ──────────────────────────────────────────────────
		{"empty input", "", "", false},
		{"only spaces", "   ", "", false},
		{"only punct", "???", "", false},
		{"only hyphens", "---", "", false},
		{"only dots", "...", "", false},
		{"only non-ASCII", "日本語", "", false},

		// ── control chars ──────────────────────────────────────────────────
		{"NUL", "foo\x00bar", "foo-bar", true},
		{"newline", "foo\nbar", "foo-bar", true},
		{"tab", "foo\tbar", "foo-bar", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := sanitizeForPath(tc.in)
			if got != tc.want || ok != tc.wantOk {
				t.Errorf("sanitizeForPath(%q) = (%q, %v), want (%q, %v)",
					tc.in, got, ok, tc.want, tc.wantOk)
			}
		})
	}
}

func TestSanitizeNamesInPassthrough(t *testing.T) {
	cases := []struct {
		name         string
		in           []string
		wantOut      []string
		wantWarnings int
	}{
		{
			name:         "no name present",
			in:           []string{"--", "fix the bug"},
			wantOut:      []string{"--", "fix the bug"},
			wantWarnings: 0,
		},
		{
			name:         "clean --name split form",
			in:           []string{"--name", "fix-bug", "--", "go"},
			wantOut:      []string{"--name", "fix-bug", "--", "go"},
			wantWarnings: 0,
		},
		{
			name:         "dirty --name split form",
			in:           []string{"--name", "foo/bar baz", "--", "go"},
			wantOut:      []string{"--name", "foo-bar-baz", "--", "go"},
			wantWarnings: 1,
		},
		{
			name:         "dirty --name= form",
			in:           []string{"--name=foo/bar", "--", "go"},
			wantOut:      []string{"--name=foo-bar", "--", "go"},
			wantWarnings: 1,
		},
		{
			name:         "dirty -n split form",
			in:           []string{"-n", "weird name!", "--"},
			wantOut:      []string{"-n", "weird-name", "--"},
			wantWarnings: 1,
		},
		{
			name:         "dirty -n= form",
			in:           []string{"-n=foo/bar"},
			wantOut:      []string{"-n=foo-bar"},
			wantWarnings: 1,
		},
		{
			name:         "all-unsafe value passes through with warning",
			in:           []string{"--name", "???", "--", "go"},
			wantOut:      []string{"--name", "???", "--", "go"},
			wantWarnings: 1,
		},
		{
			name:         "multiple names sanitized independently",
			in:           []string{"--name=foo/bar", "-n", "baz qux"},
			wantOut:      []string{"--name=foo-bar", "-n", "baz-qux"},
			wantWarnings: 2,
		},
		{
			name:         "--name at end with no value passes through untouched",
			in:           []string{"--name"},
			wantOut:      []string{"--name"},
			wantWarnings: 0,
		},
		{
			name:         "-n at end with no value passes through untouched",
			in:           []string{"-n"},
			wantOut:      []string{"-n"},
			wantWarnings: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotOut, gotWarnings := sanitizeNamesInPassthrough(tc.in)
			if !reflect.DeepEqual(gotOut, tc.wantOut) {
				t.Errorf("passthrough mismatch:\n got: %#v\nwant: %#v", gotOut, tc.wantOut)
			}
			if len(gotWarnings) != tc.wantWarnings {
				t.Errorf("warning count: got %d, want %d (warnings=%v)",
					len(gotWarnings), tc.wantWarnings, gotWarnings)
			}
		})
	}
}
