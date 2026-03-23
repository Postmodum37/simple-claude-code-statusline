package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// UsageData holds rate limit and usage data from the API or cache.
type UsageData struct {
	FetchedAt  int64        `json:"fetched_at"`
	FiveHour   *UsageWindow `json:"five_hour,omitempty"`
	SevenDay   *UsageWindow `json:"seven_day,omitempty"`
	ExtraUsage *ExtraUsage  `json:"extra_usage,omitempty"`
}

// UsageWindow represents a single rate limit window (5h or 7d).
type UsageWindow struct {
	Utilization float64 `json:"utilization"`
	ResetsAt    string  `json:"resets_at"`
}

// ExtraUsage holds extra/paid usage information.
type ExtraUsage struct {
	IsEnabled    bool    `json:"is_enabled"`
	Utilization  float64 `json:"utilization"`
	UsedCredits  float64 `json:"used_credits"`
	MonthlyLimit float64 `json:"monthly_limit"`
}

// IsStale returns true if the cache is older than ttlSeconds.
func (u *UsageData) IsStale(ttlSeconds int64) bool {
	return time.Now().Unix()-u.FetchedAt > ttlSeconds
}

// readUsageCache reads a cached UsageData from disk.
// Returns nil for nonexistent or corrupted files.
func readUsageCache(path string) *UsageData {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var cache UsageData
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil
	}
	return &cache
}

// writeUsageCache writes UsageData to disk atomically using tmpfile + rename.
func writeUsageCache(path string, data *UsageData) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "usage-cache-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(raw); err != nil {
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

// credentialsJSON is the structure of ~/.claude/.credentials.json
type credentialsJSON struct {
	ClaudeAiOauth struct {
		AccessToken string `json:"accessToken"`
	} `json:"claudeAiOauth"`
}

// getOAuthToken retrieves the OAuth access token.
// On macOS, it first tries the system keychain, then falls back to the credentials file.
// On other platforms, it reads the credentials file directly.
func getOAuthToken(credentialsPath string) (string, error) {
	if runtime.GOOS == "darwin" {
		token, err := getOAuthTokenFromKeychain()
		if err == nil && token != "" {
			return token, nil
		}
	}
	return getOAuthTokenFromFile(credentialsPath)
}

// getOAuthTokenFromKeychain tries to read the token from macOS keychain.
func getOAuthTokenFromKeychain() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "security", "find-generic-password", "-s", "Claude Code-credentials", "-w")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("keychain: %w", err)
	}

	var creds credentialsJSON
	if err := json.Unmarshal(out, &creds); err != nil {
		return "", fmt.Errorf("keychain JSON: %w", err)
	}
	if creds.ClaudeAiOauth.AccessToken == "" {
		return "", fmt.Errorf("keychain: empty access token")
	}
	return creds.ClaudeAiOauth.AccessToken, nil
}

// getOAuthTokenFromFile reads the token from a credentials JSON file.
func getOAuthTokenFromFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("credentials file: %w", err)
	}
	var creds credentialsJSON
	if err := json.Unmarshal(data, &creds); err != nil {
		return "", fmt.Errorf("credentials JSON: %w", err)
	}
	if creds.ClaudeAiOauth.AccessToken == "" {
		return "", fmt.Errorf("credentials: empty access token")
	}
	return creds.ClaudeAiOauth.AccessToken, nil
}

// convertStdinRateLimits converts stdin RateLimits to UsageData.
func convertStdinRateLimits(rl *RateLimits) *UsageData {
	data := &UsageData{
		FetchedAt: time.Now().Unix(),
	}
	if rl.FiveHour != nil {
		data.FiveHour = convertWindow(rl.FiveHour)
	}
	if rl.SevenDay != nil {
		data.SevenDay = convertWindow(rl.SevenDay)
	}
	return data
}

// convertWindow converts a stdin RateLimitWindow to a UsageWindow.
func convertWindow(w *RateLimitWindow) *UsageWindow {
	uw := &UsageWindow{}
	if w.UsedPercentage != nil {
		uw.Utilization = *w.UsedPercentage
	}
	if w.ResetsAt != nil {
		uw.ResetsAt = time.Unix(int64(*w.ResetsAt), 0).UTC().Format(time.RFC3339)
	}
	return uw
}

// usageCachePath returns the cache file path for usage data.
func usageCachePath(cacheDir string) string {
	return filepath.Join(cacheDir, "claude-statusline-usage.json")
}

const (
	usageCacheTTL      = 120 // seconds
	usageFetchTimeout  = 5 * time.Second
	usageWaitTimeout   = 200 * time.Millisecond
	usageAPIURL        = "https://api.anthropic.com/api/oauth/usage"
	usageAPIBetaHeader = "oauth-2025-04-20"
)

// GetUsageData returns usage/rate limit data using a two-phase approach:
// 1. If stdin has rate limits, use those directly (no-op WaitGroup).
// 2. If cache is fresh, use it (no-op WaitGroup).
// 3. Otherwise, start background fetch, wait up to 200ms, return whatever is available.
//
// Returns the data (possibly stale or nil) and a WaitGroup that completes
// when any background fetch finishes.
func GetUsageData(cacheDir string, stdin *StdinData) (*UsageData, *sync.WaitGroup) {
	var wg sync.WaitGroup

	// Phase 1: stdin rate limits
	if stdin.RateLimits != nil {
		data := convertStdinRateLimits(stdin.RateLimits)
		return data, &wg
	}

	// Phase 2: cache
	cachePath := usageCachePath(cacheDir)
	cached := readUsageCache(cachePath)

	if cached != nil && !cached.IsStale(usageCacheTTL) {
		return cached, &wg
	}

	// Phase 3: background fetch
	ch := make(chan *UsageData, 1)
	wg.Add(1)
	go func() {
		defer wg.Done()
		result := fetchUsageFromAPI(cachePath)
		ch <- result
	}()

	// Wait up to 200ms for fresh data
	select {
	case fresh := <-ch:
		if fresh != nil {
			return fresh, &wg
		}
		// Background fetch failed, return stale cache
		return cached, &wg
	case <-time.After(usageWaitTimeout):
		// Timed out waiting, return stale cache (may be nil)
		return cached, &wg
	}
}

// fetchUsageFromAPI fetches usage data from the API and writes it to cache.
// Returns the fetched data or nil on failure.
func fetchUsageFromAPI(cachePath string) *UsageData {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	credentialsPath := filepath.Join(homeDir, ".claude", ".credentials.json")

	token, err := getOAuthToken(credentialsPath)
	if err != nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), usageFetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", usageAPIURL, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("anthropic-beta", usageAPIBetaHeader)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	var data UsageData
	if err := json.Unmarshal(body, &data); err != nil {
		return nil
	}
	data.FetchedAt = time.Now().Unix()

	// Write cache (best effort)
	writeUsageCache(cachePath, &data)

	return &data
}
