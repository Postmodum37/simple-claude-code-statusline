package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseGitPorcelain(t *testing.T) {
	tests := []struct {
		name                         string
		output                       string
		wantAdded, wantMod, wantDel int
	}{
		{
			name:      "mixed untracked, modified, deleted",
			output:    "?? newfile.txt\n M changed.txt\n D removed.txt",
			wantAdded: 1, wantMod: 1, wantDel: 1,
		},
		{
			name:      "staged added and staged-modified",
			output:    "A  staged.txt\nAM staged-modified.txt",
			wantAdded: 2, wantMod: 0, wantDel: 0,
		},
		{
			name:      "conflict",
			output:    "UU conflict.txt",
			wantAdded: 0, wantMod: 1, wantDel: 0,
		},
		{
			name:      "empty string",
			output:    "",
			wantAdded: 0, wantMod: 0, wantDel: 0,
		},
		{
			name:      "renamed clean",
			output:    "R  old.txt -> new.txt",
			wantAdded: 1, wantMod: 0, wantDel: 0,
		},
		{
			name:      "renamed and modified",
			output:    "RM old.txt -> new.txt",
			wantAdded: 0, wantMod: 1, wantDel: 0,
		},
		{
			name:      "renamed and deleted",
			output:    "RD old.txt -> new.txt",
			wantAdded: 0, wantMod: 0, wantDel: 1,
		},
		{
			name:      "copied clean",
			output:    "C  src.txt -> dst.txt",
			wantAdded: 1, wantMod: 0, wantDel: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, m, d := parseGitPorcelain(tt.output)
			if a != tt.wantAdded || m != tt.wantMod || d != tt.wantDel {
				t.Errorf("parseGitPorcelain(%q) = (%d,%d,%d), want (%d,%d,%d)",
					tt.output, a, m, d, tt.wantAdded, tt.wantMod, tt.wantDel)
			}
		})
	}
}

func TestGitCacheReadNonexistent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.json")

	cache, err := readGitCache(path)
	if err != nil {
		t.Fatalf("readGitCache on nonexistent file: unexpected error: %v", err)
	}
	if cache != nil {
		t.Fatalf("readGitCache on nonexistent file: expected nil, got %+v", cache)
	}
}

func TestGitCacheWriteThenRead(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test-cache.json")

	original := &GitCache{
		FetchedAt: time.Now().Unix(),
		Status: GitStatus{
			Branch:   "main",
			Added:    3,
			Modified: 1,
			Deleted:  0,
			Ahead:    2,
			Behind:   0,
		},
	}

	if err := writeGitCache(path, original); err != nil {
		t.Fatalf("writeGitCache: %v", err)
	}

	got, err := readGitCache(path)
	if err != nil {
		t.Fatalf("readGitCache: %v", err)
	}
	if got == nil {
		t.Fatal("readGitCache returned nil after write")
	}
	if got.FetchedAt != original.FetchedAt {
		t.Errorf("FetchedAt = %d, want %d", got.FetchedAt, original.FetchedAt)
	}
	if got.Status.Branch != "main" {
		t.Errorf("Branch = %q, want %q", got.Status.Branch, "main")
	}
	if got.Status.Added != 3 {
		t.Errorf("Added = %d, want 3", got.Status.Added)
	}
	if got.Status.Ahead != 2 {
		t.Errorf("Ahead = %d, want 2", got.Status.Ahead)
	}
}

func TestGitCacheStale(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stale-cache.json")

	old := &GitCache{
		FetchedAt: time.Now().Add(-10 * time.Second).Unix(),
		Status: GitStatus{
			Branch: "feature",
		},
	}

	if err := writeGitCache(path, old); err != nil {
		t.Fatalf("writeGitCache: %v", err)
	}

	got, err := readGitCache(path)
	if err != nil {
		t.Fatalf("readGitCache: %v", err)
	}
	if got == nil {
		t.Fatal("readGitCache returned nil for stale cache")
	}
	if !got.IsStale(5) {
		t.Error("expected cache to be stale with 5s TTL, but IsStale(5) returned false")
	}
	if got.IsStale(15) {
		t.Error("expected cache to NOT be stale with 15s TTL, but IsStale(15) returned true")
	}
}

func TestGitCacheCorrupted(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "corrupt.json")

	if err := os.WriteFile(path, []byte("not json at all {{{"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cache, err := readGitCache(path)
	if err != nil {
		t.Fatalf("readGitCache on corrupted file: unexpected error: %v", err)
	}
	if cache != nil {
		t.Fatalf("readGitCache on corrupted file: expected nil, got %+v", cache)
	}
}

func TestGitCacheAtomicWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "atomic.json")

	cache := &GitCache{
		FetchedAt: time.Now().Unix(),
		Status:    GitStatus{Branch: "main"},
	}

	if err := writeGitCache(path, cache); err != nil {
		t.Fatalf("writeGitCache: %v", err)
	}

	// Verify the file is valid JSON
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var decoded GitCache
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("written file is not valid JSON: %v", err)
	}
}

func TestGetGitStatusEmptyDir(t *testing.T) {
	got := GetGitStatus("", t.TempDir())
	if got != nil {
		t.Errorf("GetGitStatus with empty projectDir should return nil, got %+v", got)
	}
}

func TestGetGitStatusLiveRepo(t *testing.T) {
	// This test runs against the actual repo we're in.
	// It exercises the full GetGitStatus path including cache.
	repoDir := "/Users/tomasn/Workspace/personal/simple-claude-code-statusline"
	cacheDir := t.TempDir()

	status := GetGitStatus(repoDir, cacheDir)
	if status == nil {
		t.Fatal("GetGitStatus returned nil for a valid git repo")
	}
	if status.Branch == "" {
		t.Error("Branch should not be empty for a valid git repo")
	}

	// Call again — should hit cache
	status2 := GetGitStatus(repoDir, cacheDir)
	if status2 == nil {
		t.Fatal("second GetGitStatus returned nil")
	}
	if status2.Branch != status.Branch {
		t.Errorf("cached Branch = %q, want %q", status2.Branch, status.Branch)
	}
}

func TestTruncateBranch(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"main", "main"},
		{"short-branch", "short-branch"},
		{"exactly-twenty-chars", "exactly-twenty-chars"},
		{"this-is-a-very-long-branch-name", "this-is-a-very-long\u2026"},
	}
	for _, tt := range tests {
		got := truncateBranch(tt.input, 20)
		if got != tt.want {
			t.Errorf("truncateBranch(%q, 20) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
