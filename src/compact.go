package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
)

// CompactConfig points at the files Claude Code consults for auto-compact
// configuration. All paths are optional; missing files are ignored.
type CompactConfig struct {
	// ClaudeJSONPath is the legacy global config (~/.claude.json). Claude Code
	// still honors `autoCompactEnabled` here when no settings file sets it,
	// and caches server-provided per-model auto-compact windows here under
	// `autoCompactWindowsCache`.
	ClaudeJSONPath string
	// SettingsPaths are settings.json files ordered from lowest to highest
	// precedence (user → project → local → managed). `autoCompactEnabled` and
	// `autoCompactWindow` in a higher-precedence file win.
	SettingsPaths []string
	// ModelID selects the entry in `autoCompactWindowsCache`, when present.
	ModelID string
}

// DefaultCompactConfig builds the config for the current user/project,
// mirroring Claude Code's settings precedence as of v2.1.259.
func DefaultCompactConfig(home, projectDir, modelID string) CompactConfig {
	cfg := CompactConfig{
		ClaudeJSONPath: filepath.Join(home, ".claude.json"),
		ModelID:        modelID,
	}
	cfg.SettingsPaths = append(cfg.SettingsPaths, filepath.Join(home, ".claude", "settings.json"))
	if projectDir != "" {
		cfg.SettingsPaths = append(cfg.SettingsPaths,
			filepath.Join(projectDir, ".claude", "settings.json"),
			filepath.Join(projectDir, ".claude", "settings.local.json"),
		)
	}
	switch runtime.GOOS {
	case "darwin":
		cfg.SettingsPaths = append(cfg.SettingsPaths, "/Library/Application Support/ClaudeCode/managed-settings.json")
	case "linux":
		cfg.SettingsPaths = append(cfg.SettingsPaths, "/etc/claude-code/managed-settings.json")
	}
	return cfg
}

// Auto-compact constants, verified against Claude Code v2.1.259.
const (
	compactOutputReserve = 20000  // reserved for the model's response (capped by CLAUDE_CODE_MAX_OUTPUT_TOKENS)
	compactBuffer        = 13000  // auto-compact fires this many tokens before the effective window fills
	compactWindowMin     = 100000 // smallest accepted autoCompactWindow / CLAUDE_CODE_AUTO_COMPACT_WINDOW
	compactWindowMax     = 1000000
)

// GetCompactThreshold determines the auto-compact threshold percentage for
// the given context window size, replicating Claude Code's own logic:
//
//	enabled   = !DISABLE_AUTO_COMPACT && !DISABLE_COMPACT && autoCompactEnabled (settings.json, then legacy ~/.claude.json; default true)
//	window    = min(contextWindowSize, CLAUDE_CODE_AUTO_COMPACT_WINDOW | autoCompactWindow setting | server per-model cache)
//	effective = window − min(maxOutputTokens, 20000)
//	threshold = effective − 13000   (CLAUDE_AUTOCOMPACT_PCT_OVERRIDE can only lower it)
//
// Returns (enabled, thresholdPct) where thresholdPct is 0-100 relative to the
// full contextWindowSize, i.e. the used_percentage at which auto-compact fires.
func GetCompactThreshold(contextWindowSize int, cfg CompactConfig) (bool, int) {
	if contextWindowSize <= 0 {
		return false, 0
	}

	// 1. Kill switches.
	if envTruthy(os.Getenv("DISABLE_AUTO_COMPACT")) || envTruthy(os.Getenv("DISABLE_COMPACT")) {
		return false, 0
	}

	// 2. autoCompactEnabled: settings files (highest precedence wins), then
	//    legacy ~/.claude.json, default true.
	settings := loadSettings(cfg.SettingsPaths)
	legacy := readJSONObject(cfg.ClaudeJSONPath)

	enabled := true
	if v, ok := lookupBool(settings, "autoCompactEnabled"); ok {
		enabled = v
	} else if v, ok := legacy["autoCompactEnabled"].(bool); ok {
		enabled = v
	}
	if !enabled {
		return false, 0
	}

	// 3. Auto-compact window: env var → settings → server-provided per-model
	//    cache → model's full window.
	cappedWindow := contextWindowSize
	if n, ok := envInt(os.Getenv("CLAUDE_CODE_AUTO_COMPACT_WINDOW")); ok && n > 0 {
		// Claude Code clamps the env var into [100k, 1M] before applying it.
		if n < compactWindowMin {
			n = compactWindowMin
		}
		if n > compactWindowMax {
			n = compactWindowMax
		}
		if n < cappedWindow {
			cappedWindow = n
		}
	} else if n, ok := lookupInt(settings, "autoCompactWindow"); ok && n >= compactWindowMin && n <= compactWindowMax {
		if n < cappedWindow {
			cappedWindow = n
		}
	} else if n, ok := cachedModelWindow(legacy, cfg.ModelID); ok {
		if n < cappedWindow {
			cappedWindow = n
		}
	}

	// 4. Output reserve: 20k, or less when the max output is configured lower.
	outputReserve := compactOutputReserve
	if n, ok := envInt(os.Getenv("CLAUDE_CODE_MAX_OUTPUT_TOKENS")); ok && n > 0 && n < outputReserve {
		outputReserve = n
	}
	effectiveWindow := cappedWindow - outputReserve
	if effectiveWindow <= 0 {
		return false, 0
	}

	// 5. Threshold, optionally lowered by CLAUDE_AUTOCOMPACT_PCT_OVERRIDE.
	threshold := effectiveWindow - compactBuffer
	if pct, ok := envInt(os.Getenv("CLAUDE_AUTOCOMPACT_PCT_OVERRIDE")); ok && pct >= 1 && pct <= 100 {
		if userThreshold := effectiveWindow * pct / 100; userThreshold < threshold {
			threshold = userThreshold
		}
	}
	if threshold <= 0 {
		return true, 0
	}

	// 6. Percentage against the ORIGINAL context window size, clamped.
	thresholdPct := threshold * 100 / contextWindowSize
	if thresholdPct < 0 {
		thresholdPct = 0
	}
	if thresholdPct > 100 {
		thresholdPct = 100
	}
	return true, thresholdPct
}

// --- helpers ---

// loadSettings reads each settings file and returns them highest-precedence
// first, skipping missing or malformed files.
func loadSettings(paths []string) []map[string]any {
	var out []map[string]any
	for i := len(paths) - 1; i >= 0; i-- {
		if m := readJSONObject(paths[i]); m != nil {
			out = append(out, m)
		}
	}
	return out
}

func readJSONObject(path string) map[string]any {
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var m map[string]any
	if json.Unmarshal(data, &m) != nil {
		return nil
	}
	return m
}

func lookupBool(settings []map[string]any, key string) (bool, bool) {
	for _, m := range settings {
		if v, ok := m[key].(bool); ok {
			return v, true
		}
	}
	return false, false
}

func lookupInt(settings []map[string]any, key string) (int, bool) {
	for _, m := range settings {
		if v, ok := m[key].(float64); ok {
			return int(v), true
		}
	}
	return 0, false
}

// cachedModelWindow reads Claude Code's server-provided auto-compact window
// for the model from `autoCompactWindowsCache` in ~/.claude.json. Entries are
// either a bare number or an object with a numeric `default`.
func cachedModelWindow(legacy map[string]any, modelID string) (int, bool) {
	if modelID == "" {
		return 0, false
	}
	cache, ok := legacy["autoCompactWindowsCache"].(map[string]any)
	if !ok {
		return 0, false
	}
	var n float64
	switch v := cache[modelID].(type) {
	case float64:
		n = v
	case map[string]any:
		d, ok := v["default"].(float64)
		if !ok {
			return 0, false
		}
		n = d
	default:
		return 0, false
	}
	if n < compactWindowMin || n > compactWindowMax {
		return 0, false
	}
	return int(n), true
}

func envInt(s string) (int, bool) {
	if s == "" {
		return 0, false
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return n, true
}

// envTruthy mirrors Claude Code's boolean env parsing: "1", "true", "yes", "on".
func envTruthy(s string) bool {
	switch s {
	case "1", "true", "TRUE", "True", "yes", "YES", "on", "ON":
		return true
	}
	return false
}
