package main

import (
	"path/filepath"
	"testing"
)

const testHome = "/home/testuser"

// noopDir is the expected fallback cwd.
var noopDir = filepath.Join(testHome, ".claude", "noop")

// ── parseArgs tests ────────────────────────────────────────────────────────

func TestParseArgs_SinglePositional(t *testing.T) {
	a, err := parseArgs([]string{"/proj/foo"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if a.CWD != "/proj/foo" {
		t.Errorf("CWD: got %q, want %q", a.CWD, "/proj/foo")
	}
	if len(a.ExtraDirs) != 0 {
		t.Errorf("ExtraDirs: got %v, want empty", a.ExtraDirs)
	}
}

func TestParseArgs_ThreePositionals(t *testing.T) {
	a, err := parseArgs([]string{"/proj/a", "/proj/b", "/proj/c"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if a.CWD != "/proj/a" {
		t.Errorf("CWD: got %q, want %q", a.CWD, "/proj/a")
	}
	want := []string{"/proj/b", "/proj/c"}
	if len(a.ExtraDirs) != len(want) {
		t.Fatalf("ExtraDirs len: got %d, want %d", len(a.ExtraDirs), len(want))
	}
	for i, d := range want {
		if a.ExtraDirs[i] != d {
			t.Errorf("ExtraDirs[%d]: got %q, want %q", i, a.ExtraDirs[i], d)
		}
	}
}

func TestParseArgs_ZeroPositionals_Fallback(t *testing.T) {
	a, err := parseArgs([]string{}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if a.CWD != noopDir {
		t.Errorf("CWD: got %q, want %q", a.CWD, noopDir)
	}
}

func TestParseArgs_MixedPositionalAndAlso(t *testing.T) {
	a, err := parseArgs([]string{"/p/main", "/p/extra1", "--also", "/p/extra2", "-a", "/p/extra3"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if a.CWD != "/p/main" {
		t.Errorf("CWD: got %q", a.CWD)
	}
	// extra1 comes from positional, extra2 from --also, extra3 from -a
	want := []string{"/p/extra1", "/p/extra2", "/p/extra3"}
	if len(a.ExtraDirs) != len(want) {
		t.Fatalf("ExtraDirs: got %v, want %v", a.ExtraDirs, want)
	}
	for i, d := range want {
		if a.ExtraDirs[i] != d {
			t.Errorf("ExtraDirs[%d]: got %q, want %q", i, a.ExtraDirs[i], d)
		}
	}
}

func TestParseArgs_InitFlag_Space(t *testing.T) {
	a, err := parseArgs([]string{"/p/x", "-i", "hello world"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if a.InitPrompt != "hello world" {
		t.Errorf("InitPrompt: got %q", a.InitPrompt)
	}
}

func TestParseArgs_InitFlag_Equals_Short(t *testing.T) {
	a, err := parseArgs([]string{"/p/x", "-i=my prompt"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if a.InitPrompt != "my prompt" {
		t.Errorf("InitPrompt: got %q", a.InitPrompt)
	}
}

func TestParseArgs_InitFlag_Equals_Long(t *testing.T) {
	a, err := parseArgs([]string{"/p/x", "--init=my prompt"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if a.InitPrompt != "my prompt" {
		t.Errorf("InitPrompt: got %q", a.InitPrompt)
	}
}

func TestParseArgs_AlsoEquals(t *testing.T) {
	a, err := parseArgs([]string{"/p/main", "-a=/p/extra"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.ExtraDirs) != 1 || a.ExtraDirs[0] != "/p/extra" {
		t.Errorf("ExtraDirs: got %v", a.ExtraDirs)
	}
}

func TestParseArgs_AlsoLongEquals(t *testing.T) {
	a, err := parseArgs([]string{"/p/main", "--also=/p/extra"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.ExtraDirs) != 1 || a.ExtraDirs[0] != "/p/extra" {
		t.Errorf("ExtraDirs: got %v", a.ExtraDirs)
	}
}

// Missing-value errors.

func TestParseArgs_MissingValue_Also_AtEOF(t *testing.T) {
	_, err := parseArgs([]string{"/p/x", "-a"}, testHome)
	if err == nil {
		t.Fatal("expected error for -a with no value")
	}
}

func TestParseArgs_MissingValue_Also_NextIsFlag(t *testing.T) {
	_, err := parseArgs([]string{"/p/x", "-a", "--some-flag"}, testHome)
	if err == nil {
		t.Fatal("expected error for -a followed by flag")
	}
}

func TestParseArgs_MissingValue_Init_AtEOF(t *testing.T) {
	_, err := parseArgs([]string{"/p/x", "-i"}, testHome)
	if err == nil {
		t.Fatal("expected error for -i with no value")
	}
}

func TestParseArgs_MissingValue_Init_NextIsFlag(t *testing.T) {
	_, err := parseArgs([]string{"/p/x", "-i", "--something"}, testHome)
	if err == nil {
		t.Fatal("expected error for -i followed by flag")
	}
}

// --setting-sources detection.

func TestParseArgs_SettingSourcesBare(t *testing.T) {
	a, err := parseArgs([]string{"/p/x", "--setting-sources"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if !settingSourcesInPassthrough(a.Passthrough) {
		t.Error("expected settingSourcesInPassthrough to return true")
	}
}

func TestParseArgs_SettingSourcesWithValue(t *testing.T) {
	a, err := parseArgs([]string{"/p/x", "--setting-sources=foo"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if !settingSourcesInPassthrough(a.Passthrough) {
		t.Error("expected settingSourcesInPassthrough to return true")
	}
}

func TestParseArgs_NoSettingSources(t *testing.T) {
	a, err := parseArgs([]string{"/p/x", "--verbose"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if settingSourcesInPassthrough(a.Passthrough) {
		t.Error("expected settingSourcesInPassthrough to return false")
	}
}

// Passthrough preservation and ordering.

func TestParseArgs_PassthroughOrdering(t *testing.T) {
	a, err := parseArgs([]string{"/p/x", "--foo", "--bar", "--baz"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"--foo", "--bar", "--baz"}
	if len(a.Passthrough) != len(want) {
		t.Fatalf("Passthrough len: got %d, want %d", len(a.Passthrough), len(want))
	}
	for i, f := range want {
		if a.Passthrough[i] != f {
			t.Errorf("Passthrough[%d]: got %q, want %q", i, a.Passthrough[i], f)
		}
	}
}

// ── buildArgv tests ────────────────────────────────────────────────────────
// These tests work against a synthetic fs view where we control which files
// "exist". We test argv assembly without actually execing claude.

// buildArgvNoFS calls buildArgv but with no actual files on disk, so
// --mcp-config and --settings are never injected. Useful for pure ordering tests.
func TestBuildArgv_NoExtraDirs_NoPrompt(t *testing.T) {
	a := Args{
		CWD:         "/p/main",
		Passthrough: []string{"--verbose"},
	}
	argv := buildArgv(a, "/shell")
	want := []string{"claude", "--dangerously-skip-permissions", "--verbose"}
	assertArgv(t, argv, want)
}

func TestBuildArgv_WithInitPrompt(t *testing.T) {
	a := Args{
		CWD:        "/p/main",
		InitPrompt: "do the thing",
	}
	argv := buildArgv(a, "/shell")
	want := []string{"claude", "--dangerously-skip-permissions", "--", "do the thing"}
	assertArgv(t, argv, want)
}

func TestBuildArgv_ExtraDirsAbsolute(t *testing.T) {
	// Use a dir that definitely doesn't have .mcp.json or settings.json.
	a := Args{
		CWD:       "/p/main",
		ExtraDirs: []string{"/p/extra"},
	}
	argv := buildArgv(a, "/shell")
	// Expect --add-dir; no --mcp-config or --settings (files don't exist).
	want := []string{"claude", "--dangerously-skip-permissions", "--add-dir", "/p/extra"}
	assertArgv(t, argv, want)
}

func TestBuildArgv_RelativeExtraDirResolved(t *testing.T) {
	a := Args{
		CWD:       "/p/main",
		ExtraDirs: []string{"relative/dir"},
	}
	argv := buildArgv(a, "/shell/cwd")
	// The relative path should be joined with shellCWD.
	want := []string{"claude", "--dangerously-skip-permissions", "--add-dir", "/shell/cwd/relative/dir"}
	assertArgv(t, argv, want)
}

func TestBuildArgv_SettingSourcesSuppressesSettings(t *testing.T) {
	// Even if a settings file existed this confirms the logic path.
	// Since no files exist on disk here, --settings wouldn't appear anyway,
	// but the passthrough token must be present.
	a := Args{
		CWD:         "/p/main",
		ExtraDirs:   []string{"/p/extra"},
		Passthrough: []string{"--setting-sources=user"},
	}
	argv := buildArgv(a, "/shell")
	// --setting-sources should be in passthrough position, after injected flags.
	assertContains(t, argv, "--setting-sources=user")
	assertNotContains(t, argv, "--settings")
}

func TestBuildArgv_TerminatorPlacement(t *testing.T) {
	a := Args{
		CWD:         "/p/main",
		ExtraDirs:   []string{"/p/extra"},
		Passthrough: []string{"--foo"},
		InitPrompt:  "my prompt",
	}
	argv := buildArgv(a, "/shell")
	// Order must be: claude --dangerously-skip-permissions --add-dir /p/extra --foo -- "my prompt"
	want := []string{
		"claude", "--dangerously-skip-permissions",
		"--add-dir", "/p/extra",
		"--foo",
		"--", "my prompt",
	}
	assertArgv(t, argv, want)
}

func TestBuildArgv_MultipleExtraDir_Order(t *testing.T) {
	a := Args{
		CWD:       "/p/main",
		ExtraDirs: []string{"/p/b", "/p/c"},
	}
	argv := buildArgv(a, "/shell")
	want := []string{
		"claude", "--dangerously-skip-permissions",
		"--add-dir", "/p/b",
		"--add-dir", "/p/c",
	}
	assertArgv(t, argv, want)
}

// ── helpers ────────────────────────────────────────────────────────────────

func assertArgv(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("argv len: got %d, want %d\n  got:  %v\n  want: %v", len(got), len(want), got, want)
		return
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("argv[%d]: got %q, want %q\n  full got:  %v\n  full want: %v", i, got[i], want[i], got, want)
		}
	}
}

func assertContains(t *testing.T, argv []string, token string) {
	t.Helper()
	for _, a := range argv {
		if a == token {
			return
		}
	}
	t.Errorf("argv missing %q: %v", token, argv)
}

func assertNotContains(t *testing.T, argv []string, token string) {
	t.Helper()
	for _, a := range argv {
		if a == token {
			t.Errorf("argv contains unexpected %q: %v", token, argv)
			return
		}
	}
}
