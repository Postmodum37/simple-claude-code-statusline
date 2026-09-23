package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestParseGitStatus(t *testing.T) {
	const oid = "# branch.oid 0525546fcc6792271455f9acee42de4b87b4daf4\n"
	tests := []struct {
		name   string
		output string
		want   GitStatus
	}{
		{
			name: "branch with upstream, mixed changes",
			output: oid + "# branch.head main\n# branch.upstream origin/main\n# branch.ab +2 -1\n" +
				"1 .M N... 100644 100644 100644 aaa aaa changed.txt\n" +
				"1 D. N... 100644 000000 000000 aaa 000 removed.txt\n" +
				"? newfile.txt\n",
			want: GitStatus{Branch: "main", Ahead: 2, Behind: 1, Added: 1, Modified: 1, Deleted: 1},
		},
		{
			name:   "no upstream: no ahead/behind",
			output: oid + "# branch.head feature\n1 M. N... 100644 100644 100644 aaa bbb f.txt\n",
			want:   GitStatus{Branch: "feature", Modified: 1},
		},
		{
			name:   "detached HEAD",
			output: oid + "# branch.head (detached)\n",
			want:   GitStatus{Branch: "HEAD"},
		},
		{
			name:   "initial commit on unborn branch",
			output: "# branch.oid (initial)\n# branch.head main\n? a.txt\n",
			want:   GitStatus{Branch: "main", Added: 1},
		},
		{
			name: "staged add, add then modified, add then deleted",
			output: oid + "# branch.head main\n" +
				"1 A. N... 000000 100644 100644 000 bbb a.txt\n" +
				"1 AM N... 000000 100644 100644 000 bbb b.txt\n" +
				"1 AD N... 000000 100644 000000 000 bbb c.txt\n",
			want: GitStatus{Branch: "main", Added: 3},
		},
		{
			name: "renames and copies",
			output: oid + "# branch.head main\n" +
				"2 R. N... 100644 100644 100644 aaa aaa R100 new.txt\told.txt\n" +
				"2 RM N... 100644 100644 100644 aaa aaa R90 new2.txt\told2.txt\n" +
				"2 RD N... 100644 100644 000000 aaa aaa R100 new3.txt\told3.txt\n" +
				"2 C. N... 100644 100644 100644 aaa aaa C100 dst.txt\tsrc.txt\n",
			want: GitStatus{Branch: "main", Added: 2, Modified: 1, Deleted: 1},
		},
		{
			name: "conflicts",
			output: oid + "# branch.head main\n" +
				"u UU N... 100644 100644 100644 100644 a b c both-modified.txt\n" +
				"u AA N... 000000 100644 100644 100644 0 b c both-added.txt\n" +
				"u UD N... 100644 100644 000000 100644 a b 0 deleted-by-them.txt\n",
			want: GitStatus{Branch: "main", Modified: 3},
		},
		{
			name: "intent-to-add, type change, submodule",
			output: oid + "# branch.head main\n" +
				"1 .A N... 000000 000000 100644 000 000 ita.txt\n" +
				"1 .T N... 100644 100644 120000 aaa aaa link\n" +
				"2 RT N... 100644 100644 120000 aaa aaa R100 new-link\told-file\n" +
				"1 .M S.M. 160000 160000 160000 aaa aaa sub\n",
			want: GitStatus{Branch: "main", Added: 1, Modified: 3},
		},
		{
			name:   "empty",
			output: "",
			want:   GitStatus{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseGitStatus(tt.output); got != tt.want {
				t.Errorf("parseGitStatus() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestGitCacheReadNonexistent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.json")

	if cache := readGitCache(path); cache != nil {
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

	got := readGitCache(path)
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

	got := readGitCache(path)
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

	if cache := readGitCache(path); cache != nil {
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

func TestGetGitStatusNotARepo(t *testing.T) {
	if got := GetGitStatus(t.TempDir(), t.TempDir()); got != nil {
		t.Errorf("GetGitStatus outside a repo should return nil, got %+v", got)
	}
}

func TestGetGitStatusRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	repo := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", repo, "-c", "user.name=t", "-c", "user.email=t@t"}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(repo, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	git("init", "-q", "-b", "main")
	write("tracked.txt", "a\n")
	write("doomed.txt", "b\n")
	git("add", ".")
	git("commit", "-q", "-m", "init")
	write("tracked.txt", "a\nb\n")
	write("new.txt", "n\n")
	if err := os.Remove(filepath.Join(repo, "doomed.txt")); err != nil {
		t.Fatal(err)
	}

	cacheDir := t.TempDir()
	want := GitStatus{Branch: "main", Added: 1, Modified: 1, Deleted: 1}
	if got := GetGitStatus(repo, cacheDir); got == nil || *got != want {
		t.Fatalf("GetGitStatus = %+v, want %+v", got, want)
	}

	// Second call within the TTL is served from cache even though the tree changed.
	write("another.txt", "x\n")
	if got := GetGitStatus(repo, cacheDir); got == nil || *got != want {
		t.Errorf("cached GetGitStatus = %+v, want %+v", got, want)
	}

	// When `git status` fails (here: corrupt index; in practice also a timeout on
	// a large repo), the branch alone still shows.
	if err := os.WriteFile(filepath.Join(repo, ".git", "index"), []byte("garbage"), 0644); err != nil {
		t.Fatal(err)
	}
	if got := GetGitStatus(repo, t.TempDir()); got == nil || *got != (GitStatus{Branch: "main"}) {
		t.Errorf("GetGitStatus with failing status = %+v, want branch only", got)
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
