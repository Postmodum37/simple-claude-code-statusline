package main

import (
	"encoding/json"
	"os"
	"strconv"
)

// GetCompactThreshold determines the auto-compact threshold percentage for
// the given context window size. It reads the claude.json config to check
// if auto-compact is enabled, and respects environment variable overrides.
//
// Returns (enabled, thresholdPct) where thresholdPct is 0-100, representing
// the used_percentage at which auto-compact would trigger.
func GetCompactThreshold(contextWindowSize int, claudeJSONPath string) (bool, int) {
	// 1. Read config to check if auto-compact is enabled (default: true)
	autoCompactEnabled := true
	if data, err := os.ReadFile(claudeJSONPath); err == nil {
		var config map[string]any
		if json.Unmarshal(data, &config) == nil {
			if v, ok := config["autoCompactEnabled"]; ok {
				if b, ok := v.(bool); ok {
					autoCompactEnabled = b
				}
			}
		}
	}

	// 2. If disabled, return immediately
	if !autoCompactEnabled {
		return false, 0
	}

	// 3. Check CLAUDE_CODE_AUTO_COMPACT_WINDOW env var
	cappedWindow := contextWindowSize
	if envVal := os.Getenv("CLAUDE_CODE_AUTO_COMPACT_WINDOW"); envVal != "" {
		if n, err := strconv.Atoi(envVal); err == nil && n > 0 {
			if n < cappedWindow {
				cappedWindow = n
			}
		}
	}

	// 4-6. Calculate effective window
	const outputReserve = 20000
	effectiveWindow := cappedWindow - outputReserve
	if effectiveWindow <= 0 {
		return false, 0
	}

	// 7. Default threshold
	defaultThreshold := effectiveWindow - 13000

	// 8-9. Check CLAUDE_AUTOCOMPACT_PCT_OVERRIDE
	threshold := defaultThreshold
	if envVal := os.Getenv("CLAUDE_AUTOCOMPACT_PCT_OVERRIDE"); envVal != "" {
		if pct, err := strconv.Atoi(envVal); err == nil && pct >= 1 && pct <= 100 {
			userThreshold := effectiveWindow * pct / 100
			if userThreshold < defaultThreshold {
				threshold = userThreshold
			}
		}
	}

	// 10. If threshold <= 0, return enabled with 0%
	if threshold <= 0 {
		return true, 0
	}

	// 11. Calculate percentage against ORIGINAL context window size
	thresholdPct := threshold * 100 / contextWindowSize

	// 12. Clamp to 0-100
	if thresholdPct < 0 {
		thresholdPct = 0
	}
	if thresholdPct > 100 {
		thresholdPct = 100
	}

	// 13. Return
	return true, thresholdPct
}
