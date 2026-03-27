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
	usageData := GetUsageData(stdin)
	gitData := GetGitStatus(stdin.Workspace.ProjectDir, cacheDir)
	compactEnabled, compactPct := GetCompactThreshold(stdin.ContextWindow.ContextWindowSize, claudeJSONPath)

	// Render to stdout
	Render(os.Stdout, stdin, gitData, usageData, CompactInfo{
		Enabled:      compactEnabled,
		ThresholdPct: compactPct,
	})
}
