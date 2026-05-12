package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// Args holds the result of parsing fnclaude's own argv.
type Args struct {
	// CWD is the directory claude will be launched in (first positional, or
	// ~/.claude/noop when no positionals given).
	CWD string

	// ExtraDirs collects positional[1:] and all -A/--also values, in order.
	ExtraDirs []string

	// Passthrough is everything else, preserved in order, to be forwarded to
	// claude verbatim.
	Passthrough []string
}

// parseArgs parses os.Args[1:] and returns the structured result.
// home is the value to use for the noop fallback (typically os.UserHomeDir()).
func parseArgs(argv []string, home string) (Args, error) {
	var firstPath string
	var extraDirs []string
	var passthrough []string

	inFlags := false // once true, non-flag tokens go to passthrough
	firstPathSet := false

	i := 0
	for i < len(argv) {
		arg := argv[i]

		// ── Positional collection (before first flag-shaped token) ───────────
		if !inFlags && !strings.HasPrefix(arg, "-") {
			if !firstPathSet {
				firstPath = arg
				firstPathSet = true
			} else {
				extraDirs = append(extraDirs, arg)
			}
			i++
			continue
		}

		// Anything from here on: we are in flag territory.
		inFlags = true

		// ── -A / --also ──────────────────────────────────────────────────────
		// Supported forms: -A <val>, -A=<val>, --also <val>, --also=<val>
		if arg == "-A" || arg == "--also" {
			if i+1 >= len(argv) || strings.HasPrefix(argv[i+1], "-") {
				which := arg
				if i+1 < len(argv) {
					which = fmt.Sprintf("%s %s", arg, argv[i+1])
				}
				return Args{}, fmt.Errorf("fnclaude: %s requires a directory argument", which)
			}
			extraDirs = append(extraDirs, argv[i+1])
			i += 2
			continue
		}
		if strings.HasPrefix(arg, "-A=") {
			val := arg[len("-A="):]
			if val == "" {
				return Args{}, fmt.Errorf("fnclaude: -A= requires a directory argument")
			}
			extraDirs = append(extraDirs, val)
			i++
			continue
		}
		if strings.HasPrefix(arg, "--also=") {
			val := arg[len("--also="):]
			if val == "" {
				return Args{}, fmt.Errorf("fnclaude: --also= requires a directory argument")
			}
			extraDirs = append(extraDirs, val)
			i++
			continue
		}

		// ── Everything else: passthrough ─────────────────────────────────────
		passthrough = append(passthrough, arg)
		i++
	}

	// CWD fallback.
	cwd := filepath.Join(home, ".claude", "noop")
	if firstPathSet {
		cwd = firstPath
	}

	return Args{
		CWD:       cwd,
		ExtraDirs: extraDirs,
		Passthrough: passthrough,
	}, nil
}

// settingSourcesInPassthrough returns true if any passthrough token is
// --setting-sources or starts with --setting-sources=.
func settingSourcesInPassthrough(passthrough []string) bool {
	for _, t := range passthrough {
		if t == "--setting-sources" || strings.HasPrefix(t, "--setting-sources=") {
			return true
		}
	}
	return false
}

// buildArgv constructs the argv slice to exec claude with, given the parsed
// args and the user's shell cwd (used to resolve relative extra-dir paths).
// shellCWD is the process working directory at fnclaude startup (os.Getwd()).
func buildArgv(a Args, shellCWD string) []string {
	suppressSettings := settingSourcesInPassthrough(a.Passthrough)

	argv := []string{"claude", "--dangerously-skip-permissions"}

	// Inject --add-dir (and optional --mcp-config / --settings) for each
	// extra dir. Paths are resolved relative to the user's shell cwd.
	for _, d := range a.ExtraDirs {
		if !filepath.IsAbs(d) {
			d = filepath.Join(shellCWD, d)
		}
		argv = append(argv, "--add-dir", d)

		mcpConfig := filepath.Join(d, ".mcp.json")
		if _, err := os.Stat(mcpConfig); err == nil {
			argv = append(argv, "--mcp-config", mcpConfig)
		}

		if !suppressSettings {
			settings := filepath.Join(d, ".claude", "settings.json")
			if _, err := os.Stat(settings); err == nil {
				argv = append(argv, "--settings", settings)
			}
		}
	}

	argv = append(argv, a.Passthrough...)

	return argv
}

func run() int {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "fnclaude: cannot determine home directory: %v\n", err)
		return 1
	}

	a, err := parseArgs(os.Args[1:], home)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	shellCWD, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "fnclaude: cannot determine working directory: %v\n", err)
		return 1
	}

	// Resolve the launch cwd relative to the shell cwd.
	launchCWD := a.CWD
	if !filepath.IsAbs(launchCWD) {
		launchCWD = filepath.Join(shellCWD, launchCWD)
	}

	argv := buildArgv(a, shellCWD)

	claudeBin, err := exec.LookPath("claude")
	if err != nil {
		fmt.Fprintf(os.Stderr, "fnclaude: claude not found in PATH: %v\n", err)
		return 1
	}

	cmd := exec.Command(claudeBin, argv[1:]...)
	cmd.Dir = launchCWD
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				return status.ExitStatus()
			}
		}
		// Non-exit error (e.g. signal death) — return 1.
		return 1
	}
	return 0
}

func main() {
	os.Exit(run())
}
