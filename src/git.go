package main

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// GitStatus holds parsed git state for display.
type GitStatus struct {
	Branch   string `json:"branch"`
	Worktree string `json:"worktree"`
	Added    int    `json:"added"`
	Modified int    `json:"modified"`
	Deleted  int    `json:"deleted"`
	Ahead    int    `json:"ahead"`
	Behind   int    `json:"behind"`
}

// GitCache wraps a GitStatus with a fetch timestamp for staleness checks.
type GitCache struct {
	FetchedAt int64     `json:"fetched_at"`
	Status    GitStatus `json:"status"`
}

// IsStale returns true if the cache is older than ttlSeconds.
func (c *GitCache) IsStale(ttlSeconds int64) bool {
	return time.Now().Unix()-c.FetchedAt > ttlSeconds
}

// parseGitPorcelain parses `git status --porcelain` output and returns
// counts of added, modified, and deleted files. Matches the bash script logic.
func parseGitPorcelain(output string) (added, modified, deleted int) {
	if output == "" {
		return 0, 0, 0
	}
	for _, line := range strings.Split(output, "\n") {
		if len(line) < 2 {
			continue
		}
		status := line[:2]
		switch status {
		// Untracked
		case "??":
			added++
		// Added (staged)
		case "A ", "AM", "AD":
			added++
		// Modified (various combinations)
		case " M", "M ", "MM", "RM", "CM":
			modified++
		// Deleted
		case " D", "D ", "MD", "RD", "CD":
			deleted++
		// Renamed/Copied clean (target is new)
		case "R ", "C ":
			added++
		// Unmerged/conflict states
		case "UU", "AA", "DD", "AU", "UA", "DU", "UD":
			modified++
		}
	}
	return
}

// readGitCache reads a cached GitCache from disk.
// Returns nil, nil for nonexistent or corrupted files.
func readGitCache(path string) (*GitCache, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, nil
	}
	var cache GitCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, nil
	}
	return &cache, nil
}

// writeGitCache writes a GitCache to disk atomically using tmpfile + rename.
func writeGitCache(path string, cache *GitCache) error {
	data, err := json.Marshal(cache)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "git-cache-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, path)
}

// truncateBranch truncates a branch name to maxLen runes, appending "…" if truncated.
func truncateBranch(branch string, maxLen int) string {
	runes := []rune(branch)
	if len(runes) <= maxLen {
		return branch
	}
	return string(runes[:maxLen-1]) + "\u2026"
}

// gitCachePath returns the cache file path for a given project directory.
func gitCachePath(projectDir, cacheDir string) string {
	checksum := crc32.ChecksumIEEE([]byte(projectDir))
	return filepath.Join(cacheDir, fmt.Sprintf("claude-statusline-git-%08x.json", checksum))
}

// runGit executes a git command with a 1-second timeout and --no-optional-locks.
// Returns trimmed stdout, or empty string on error.
func runGit(projectDir string, args ...string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	fullArgs := append([]string{"--no-optional-locks", "-C", projectDir}, args...)
	cmd := exec.CommandContext(ctx, "git", fullArgs...)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// GetGitStatus returns the current git status for projectDir, using a file cache
// in cacheDir with a 5-second TTL. Returns nil if projectDir is empty or not a git repo.
func GetGitStatus(projectDir, cacheDir string) *GitStatus {
	if projectDir == "" {
		return nil
	}

	cachePath := gitCachePath(projectDir, cacheDir)

	// Try cache first
	if cached, _ := readGitCache(cachePath); cached != nil && !cached.IsStale(5) {
		return &cached.Status
	}

	// Verify this is a git repo
	if runGit(projectDir, "rev-parse", "--git-dir") == "" {
		return nil
	}

	// Get branch name
	branch := runGit(projectDir, "rev-parse", "--abbrev-ref", "HEAD")
	branch = truncateBranch(branch, 20)

	// Parse porcelain status
	porcelain := runGit(projectDir, "status", "--porcelain")
	added, modified, deleted := parseGitPorcelain(porcelain)

	// Get ahead/behind counts
	var ahead, behind int
	leftRight := runGit(projectDir, "rev-list", "--left-right", "--count", "@{upstream}...HEAD")
	if leftRight != "" {
		parts := strings.Fields(leftRight)
		if len(parts) == 2 {
			behind, _ = strconv.Atoi(parts[0])
			ahead, _ = strconv.Atoi(parts[1])
		}
	}

	// Detect worktree: .git is a file (not directory) in linked worktrees
	var worktree string
	gitDotPath := filepath.Join(projectDir, ".git")
	if info, err := os.Stat(gitDotPath); err == nil && !info.IsDir() {
		worktree = filepath.Base(projectDir)
	}

	status := GitStatus{
		Branch:   branch,
		Worktree: worktree,
		Added:    added,
		Modified: modified,
		Deleted:  deleted,
		Ahead:    ahead,
		Behind:   behind,
	}

	// Write cache (best effort)
	cache := &GitCache{
		FetchedAt: time.Now().Unix(),
		Status:    status,
	}
	writeGitCache(cachePath, cache)

	return &status
}
