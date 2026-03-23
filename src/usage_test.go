package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// --- Cache read/write tests ---

func TestUsageCacheReadNonexistent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.json")

	got := readUsageCache(path)
	if got != nil {
		t.Fatalf("readUsageCache on nonexistent file: expected nil, got %+v", got)
	}
}

func TestUsageCacheWriteThenRead(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "usage-cache.json")

	now := time.Now().Unix()
	original := &UsageData{
		FetchedAt: now,
		FiveHour: &UsageWindow{
			Utilization: 42.0,
			ResetsAt:    "2026-03-23T18:00:00Z",
		},
		SevenDay: &UsageWindow{
			Utilization: 15.0,
			ResetsAt:    "2026-03-28T00:00:00Z",
		},
		ExtraUsage: &ExtraUsage{
			IsEnabled:    true,
			Utilization:  5.0,
			UsedCredits:  12.0,
			MonthlyLimit: 200.0,
		},
	}

	if err := writeUsageCache(path, original); err != nil {
		t.Fatalf("writeUsageCache: %v", err)
	}

	got := readUsageCache(path)
	if got == nil {
		t.Fatal("readUsageCache returned nil after write")
	}
	if got.FetchedAt != now {
		t.Errorf("FetchedAt = %d, want %d", got.FetchedAt, now)
	}
	if got.FiveHour == nil {
		t.Fatal("FiveHour is nil")
	}
	if got.FiveHour.Utilization != 42.0 {
		t.Errorf("FiveHour.Utilization = %f, want 42.0", got.FiveHour.Utilization)
	}
	if got.FiveHour.ResetsAt != "2026-03-23T18:00:00Z" {
		t.Errorf("FiveHour.ResetsAt = %q, want %q", got.FiveHour.ResetsAt, "2026-03-23T18:00:00Z")
	}
	if got.SevenDay == nil {
		t.Fatal("SevenDay is nil")
	}
	if got.SevenDay.Utilization != 15.0 {
		t.Errorf("SevenDay.Utilization = %f, want 15.0", got.SevenDay.Utilization)
	}
	if got.ExtraUsage == nil {
		t.Fatal("ExtraUsage is nil")
	}
	if !got.ExtraUsage.IsEnabled {
		t.Error("ExtraUsage.IsEnabled = false, want true")
	}
	if got.ExtraUsage.UsedCredits != 12.0 {
		t.Errorf("ExtraUsage.UsedCredits = %f, want 12.0", got.ExtraUsage.UsedCredits)
	}
}

func TestUsageCacheCorruptedJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "corrupt.json")

	if err := os.WriteFile(path, []byte("not json at all {{{"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	got := readUsageCache(path)
	if got != nil {
		t.Fatalf("readUsageCache on corrupted file: expected nil, got %+v", got)
	}
}

func TestUsageCacheIsStale(t *testing.T) {
	t.Run("not stale within TTL", func(t *testing.T) {
		data := &UsageData{
			FetchedAt: time.Now().Add(-60 * time.Second).Unix(),
		}
		if data.IsStale(120) {
			t.Error("expected cache with FetchedAt 60s ago to NOT be stale with 120s TTL")
		}
	})

	t.Run("stale beyond TTL", func(t *testing.T) {
		data := &UsageData{
			FetchedAt: time.Now().Add(-121 * time.Second).Unix(),
		}
		if !data.IsStale(120) {
			t.Error("expected cache with FetchedAt 121s ago to be stale with 120s TTL")
		}
	})
}

func TestUsageCacheAtomicWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "atomic.json")

	data := &UsageData{
		FetchedAt: time.Now().Unix(),
		FiveHour:  &UsageWindow{Utilization: 10.0, ResetsAt: "2026-03-23T18:00:00Z"},
	}

	if err := writeUsageCache(path, data); err != nil {
		t.Fatalf("writeUsageCache: %v", err)
	}

	// Verify the file is valid JSON
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var decoded UsageData
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("written file is not valid JSON: %v", err)
	}
}

// --- OAuth token retrieval tests ---
// These test getOAuthTokenFromFile directly to avoid keychain interference on macOS.

func TestGetOAuthTokenFromFile(t *testing.T) {
	dir := t.TempDir()
	credPath := filepath.Join(dir, "credentials.json")

	content := `{"claudeAiOauth": {"accessToken": "test-token-abc123"}}`
	if err := os.WriteFile(credPath, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	token, err := getOAuthTokenFromFile(credPath)
	if err != nil {
		t.Fatalf("getOAuthTokenFromFile: unexpected error: %v", err)
	}
	if token != "test-token-abc123" {
		t.Errorf("token = %q, want %q", token, "test-token-abc123")
	}
}

func TestGetOAuthTokenFileNotFound(t *testing.T) {
	dir := t.TempDir()
	credPath := filepath.Join(dir, "nonexistent.json")

	_, err := getOAuthTokenFromFile(credPath)
	if err == nil {
		t.Fatal("getOAuthTokenFromFile: expected error for nonexistent file, got nil")
	}
}

func TestGetOAuthTokenMalformedJSON(t *testing.T) {
	dir := t.TempDir()
	credPath := filepath.Join(dir, "credentials.json")

	if err := os.WriteFile(credPath, []byte("not json"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := getOAuthTokenFromFile(credPath)
	if err == nil {
		t.Fatal("getOAuthTokenFromFile: expected error for malformed JSON, got nil")
	}
}

func TestGetOAuthTokenMissingField(t *testing.T) {
	dir := t.TempDir()
	credPath := filepath.Join(dir, "credentials.json")

	content := `{"claudeAiOauth": {}}`
	if err := os.WriteFile(credPath, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := getOAuthTokenFromFile(credPath)
	if err == nil {
		t.Fatal("getOAuthTokenFromFile: expected error for missing accessToken, got nil")
	}
}

// --- GetUsageData orchestration tests ---

func TestGetUsageDataFromStdin(t *testing.T) {
	cacheDir := t.TempDir()

	fiveHourPct := 42.0
	fiveHourResets := float64(time.Date(2026, 3, 23, 18, 0, 0, 0, time.UTC).Unix())
	sevenDayPct := 15.0
	sevenDayResets := float64(time.Date(2026, 3, 28, 0, 0, 0, 0, time.UTC).Unix())

	stdin := &StdinData{
		RateLimits: &RateLimits{
			FiveHour: &RateLimitWindow{
				UsedPercentage: &fiveHourPct,
				ResetsAt:       &fiveHourResets,
			},
			SevenDay: &RateLimitWindow{
				UsedPercentage: &sevenDayPct,
				ResetsAt:       &sevenDayResets,
			},
		},
	}

	data, wg := GetUsageData(cacheDir, stdin)
	wg.Wait()

	if data == nil {
		t.Fatal("GetUsageData returned nil")
	}
	if data.FiveHour == nil {
		t.Fatal("FiveHour is nil")
	}
	if data.FiveHour.Utilization != 42.0 {
		t.Errorf("FiveHour.Utilization = %f, want 42.0", data.FiveHour.Utilization)
	}
	if data.SevenDay == nil {
		t.Fatal("SevenDay is nil")
	}
	if data.SevenDay.Utilization != 15.0 {
		t.Errorf("SevenDay.Utilization = %f, want 15.0", data.SevenDay.Utilization)
	}
}

func TestGetUsageDataFromFreshCache(t *testing.T) {
	cacheDir := t.TempDir()
	cachePath := filepath.Join(cacheDir, "claude-statusline-usage.json")

	cached := &UsageData{
		FetchedAt: time.Now().Unix(),
		FiveHour: &UsageWindow{
			Utilization: 55.0,
			ResetsAt:    "2026-03-23T18:00:00Z",
		},
	}
	if err := writeUsageCache(cachePath, cached); err != nil {
		t.Fatalf("writeUsageCache: %v", err)
	}

	stdin := &StdinData{} // no rate limits in stdin

	data, wg := GetUsageData(cacheDir, stdin)
	wg.Wait()

	if data == nil {
		t.Fatal("GetUsageData returned nil with fresh cache")
	}
	if data.FiveHour == nil || data.FiveHour.Utilization != 55.0 {
		t.Errorf("expected cached FiveHour.Utilization=55.0, got %+v", data.FiveHour)
	}
}

func TestGetUsageDataFromStaleCache(t *testing.T) {
	cacheDir := t.TempDir()
	cachePath := filepath.Join(cacheDir, "claude-statusline-usage.json")

	stale := &UsageData{
		FetchedAt: time.Now().Add(-200 * time.Second).Unix(),
		FiveHour: &UsageWindow{
			Utilization: 33.0,
			ResetsAt:    "2026-03-23T18:00:00Z",
		},
	}
	if err := writeUsageCache(cachePath, stale); err != nil {
		t.Fatalf("writeUsageCache: %v", err)
	}

	stdin := &StdinData{} // no rate limits, no valid credentials either

	data, wg := GetUsageData(cacheDir, stdin)

	// The function should return the stale data for immediate render
	if data == nil {
		t.Fatal("GetUsageData returned nil with stale cache (should return stale data)")
	}
	if data.FiveHour == nil || data.FiveHour.Utilization != 33.0 {
		t.Errorf("expected stale FiveHour.Utilization=33.0, got %+v", data.FiveHour)
	}

	// The WaitGroup should eventually complete (background fetch will fail
	// due to no credentials, but goroutine should still finish)
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		// OK
	case <-time.After(10 * time.Second):
		t.Fatal("WaitGroup did not complete within timeout")
	}
}

func TestGetUsageDataNoCacheNoStdin(t *testing.T) {
	cacheDir := t.TempDir()
	stdin := &StdinData{}

	data, wg := GetUsageData(cacheDir, stdin)

	// With no cache and no stdin rate limits, the background fetch will
	// attempt but fail (no credentials). We should still get a completed WaitGroup.
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		// OK
	case <-time.After(10 * time.Second):
		t.Fatal("WaitGroup did not complete within timeout")
	}

	// data may be nil (no cache, background fetch failed)
	_ = data
}

// --- Stdin rate limit conversion test ---

func TestConvertStdinRateLimits(t *testing.T) {
	fiveHourPct := 42.5
	fiveHourResets := float64(time.Date(2026, 3, 23, 18, 0, 0, 0, time.UTC).Unix())

	rl := &RateLimits{
		FiveHour: &RateLimitWindow{
			UsedPercentage: &fiveHourPct,
			ResetsAt:       &fiveHourResets,
		},
	}

	data := convertStdinRateLimits(rl)
	if data == nil {
		t.Fatal("convertStdinRateLimits returned nil")
	}
	if data.FiveHour == nil {
		t.Fatal("FiveHour is nil")
	}
	if data.FiveHour.Utilization != 42.5 {
		t.Errorf("FiveHour.Utilization = %f, want 42.5", data.FiveHour.Utilization)
	}
	if data.FiveHour.ResetsAt != "2026-03-23T18:00:00Z" {
		t.Errorf("FiveHour.ResetsAt = %q, want %q", data.FiveHour.ResetsAt, "2026-03-23T18:00:00Z")
	}
	if data.SevenDay != nil {
		t.Error("SevenDay should be nil when not provided")
	}
}

// --- WaitGroup helper test ---

func TestNoOpWaitGroup(t *testing.T) {
	var wg sync.WaitGroup
	// A zero-value WaitGroup should be immediately done
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		// OK
	case <-time.After(1 * time.Second):
		t.Fatal("no-op WaitGroup did not complete immediately")
	}
}
