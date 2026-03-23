package main

import (
	"os"
	"path/filepath"
)

func main() {
	stdin, err := ParseStdin(os.Stdin)
	if err != nil {
		os.Exit(0) // silent failure
	}

	cacheDir := os.Getenv("CLAUDE_CODE_TMPDIR")
	if cacheDir == "" {
		cacheDir = os.TempDir()
	}

	// Gather data
	claudeJSONPath := filepath.Join(os.Getenv("HOME"), ".claude.json")
	usageData, fetchWg := GetUsageData(cacheDir, stdin)
	gitData := GetGitStatus(stdin.Workspace.ProjectDir, cacheDir)
	compactEnabled, compactPct := GetCompactThreshold(stdin.ContextWindow.ContextWindowSize, claudeJSONPath)

	// Phase 1: Render to stdout
	Render(os.Stdout, stdin, gitData, usageData, CompactInfo{
		Enabled:      compactEnabled,
		ThresholdPct: compactPct,
	})

	// Phase 2: Close stdout (Claude Code gets output), wait for background fetch
	os.Stdout.Close()
	fetchWg.Wait()
}
