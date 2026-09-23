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

// parseGitStatus parses `git status --porcelain=v2 --branch` output: branch
// name and ahead/behind from the "# branch.*" headers, file counts from the
// entries. Detached HEAD is reported as "HEAD".
func parseGitStatus(output string) GitStatus {
	var s GitStatus
	for _, line := range strings.Split(output, "\n") {
		switch {
		case strings.HasPrefix(line, "# branch.head "):
			s.Branch = strings.TrimPrefix(line, "# branch.head ")
			if s.Branch == "(detached)" {
				s.Branch = "HEAD"
			}
		case strings.HasPrefix(line, "# branch.ab "):
			// "# branch.ab +<ahead> -<behind>"
			if f := strings.Fields(line); len(f) == 4 {
				s.Ahead, _ = strconv.Atoi(strings.TrimPrefix(f[2], "+"))
				s.Behind, _ = strconv.Atoi(strings.TrimPrefix(f[3], "-"))
			}
		case strings.HasPrefix(line, "? "):
			s.Added++
		case strings.HasPrefix(line, "u "):
			s.Modified++ // unmerged / conflict
		case len(line) >= 4 && (line[0] == '1' || line[0] == '2') && line[1] == ' ':
			// Ordinary ("1") or renamed/copied ("2") entry; XY at [2:4], "." = unmodified.
			x, y := line[2], line[3]
			switch {
			case x == 'A' || y == 'A':
				s.Added++
			case x == 'D' || y == 'D':
				s.Deleted++
			case (x == 'R' || x == 'C') && y == '.':
				s.Added++ // clean rename/copy: the target is new
			default:
				s.Modified++
			}
		}
	}
	return s
}

// readGitCache reads a cached GitCache from disk. Returns nil for
// nonexistent or corrupted files.
func readGitCache(path string) *GitCache {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var cache GitCache
	if json.Unmarshal(data, &cache) != nil {
		return nil
	}
	return &cache
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
	return string(runes[:maxLen-1]) + "…"
}

// gitCachePath returns the cache file path for a given project directory.
func gitCachePath(projectDir, cacheDir string) string {
	checksum := crc32.ChecksumIEEE([]byte(projectDir))
	return filepath.Join(cacheDir, fmt.Sprintf("claude-statusline-git-%08x.json", checksum))
}

// runGit executes a git command with a 1-second timeout and --no-optional-locks.
// Returns stdout, or empty string on error.
func runGit(projectDir string, args ...string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	fullArgs := append([]string{"--no-optional-locks", "-C", projectDir}, args...)
	out, err := exec.CommandContext(ctx, "git", fullArgs...).Output()
	if err != nil {
		return ""
	}
	return string(out)
}

// GetGitStatus returns the current git status for projectDir, using a file cache
// in cacheDir with a 5-second TTL. Returns nil if projectDir is empty or not a git repo.
func GetGitStatus(projectDir, cacheDir string) *GitStatus {
	if projectDir == "" {
		return nil
	}

	cachePath := gitCachePath(projectDir, cacheDir)
	if cached := readGitCache(cachePath); cached != nil && !cached.IsStale(5) {
		return &cached.Status
	}

	// One command covers branch, ahead/behind, and file status; inside a repo
	// it always prints the "# branch.oid" header. Empty output means it failed:
	// either not a repo, or too slow (large repo hitting the timeout). Fall back
	// to the branch alone so the git segment still shows, and cache that too.
	var status GitStatus
	if out := runGit(projectDir, "status", "--porcelain=v2", "--branch"); out != "" {
		status = parseGitStatus(out)
	} else {
		status.Branch = strings.TrimSpace(runGit(projectDir, "rev-parse", "--abbrev-ref", "HEAD"))
		if status.Branch == "" {
			return nil // not a git repo
		}
	}
	status.Branch = truncateBranch(status.Branch, 20)

	// Write cache (best effort)
	writeGitCache(cachePath, &GitCache{FetchedAt: time.Now().Unix(), Status: status})

	return &status
}
