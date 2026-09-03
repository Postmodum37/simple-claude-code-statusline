package main

import "os"

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
	usageData := GetUsageData(stdin)
	gitData := GetGitStatus(stdin.Workspace.ProjectDir, cacheDir)
	compactCfg := DefaultCompactConfig(os.Getenv("HOME"), stdin.Workspace.ProjectDir, stdin.Model.ID)
	compactEnabled, compactPct := GetCompactThreshold(stdin.ContextWindow.ContextWindowSize, compactCfg)

	// Render to stdout
	Render(os.Stdout, stdin, gitData, usageData, CompactInfo{
		Enabled:      compactEnabled,
		ThresholdPct: compactPct,
	})
}
