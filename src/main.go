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

	git := GetGitStatus(stdin.Workspace.ProjectDir, cacheDir)
	compactCfg := DefaultCompactConfig(os.Getenv("HOME"), stdin.Workspace.ProjectDir, stdin.Model.ID)
	compact := GetCompactThreshold(stdin.ContextWindow.ContextWindowSize, compactCfg)

	Render(os.Stdout, stdin, git, compact)
}
