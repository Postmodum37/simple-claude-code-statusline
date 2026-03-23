package main

import "strings"

// ModelDisplayName extracts a human-readable model name from a Claude model ID.
//
// Formats handled:
//   - New: claude-{model}-{major}[-{minor}][-{date}]  e.g. "claude-opus-4-6", "claude-sonnet-4-20250514"
//   - Old: claude-{major}-{minor}-{model}[-{date}]    e.g. "claude-3-5-sonnet-20241022"
//   - Context suffix stripped: "claude-opus-4-6[1m]"   → same as "claude-opus-4-6"
//
// Falls back to the first word of displayName when the model ID is empty or
// contains no recognized family name.
func ModelDisplayName(id, displayName string) string {
	// Detect family from model ID.
	var family string
	switch {
	case strings.Contains(id, "opus"):
		family = "Opus"
	case strings.Contains(id, "sonnet"):
		family = "Sonnet"
	case strings.Contains(id, "haiku"):
		family = "Haiku"
	default:
		// Fallback: first word of displayName.
		if w := strings.Fields(displayName); len(w) > 0 {
			return w[0]
		}
		return ""
	}

	// Strip context suffix like [1m] before version parsing.
	clean := id
	if idx := strings.Index(clean, "["); idx != -1 {
		clean = clean[:idx]
	}

	version := extractVersion(clean)
	if version != "" {
		return family + " " + version
	}
	return family
}

// extractVersion pulls a version string (e.g. "4.6", "4", "3.5") from a
// cleaned model ID (no [...] suffix).
func extractVersion(id string) string {
	// Remove "claude-" prefix.
	rest := strings.TrimPrefix(id, "claude-")
	if rest == id {
		// No "claude-" prefix — can't parse.
		return ""
	}

	// Check old format: starts with digit (e.g. "3-5-sonnet-20241022").
	if len(rest) > 0 && rest[0] >= '0' && rest[0] <= '9' {
		parts := strings.SplitN(rest, "-", 3) // ["3", "5", "sonnet-20241022"]
		if len(parts) >= 2 && isDigits(parts[0]) && isDigits(parts[1]) {
			return parts[0] + "." + parts[1]
		}
		return ""
	}

	// New format: remove "model-" prefix to get version segment.
	// e.g. "opus-4-5-20251101" → after removing "opus-" → "4-5-20251101"
	dashIdx := strings.Index(rest, "-")
	if dashIdx < 0 {
		return ""
	}
	versionPart := rest[dashIdx+1:] // "4-5-20251101" or "4-20250514" or "4-6"

	parts := strings.SplitN(versionPart, "-", 3) // ["4","5","20251101"] or ["4","20250514"]
	if len(parts) == 0 || !isDigits(parts[0]) {
		return ""
	}
	major := parts[0]

	if len(parts) >= 2 && isDigits(parts[1]) && len(parts[1]) <= 2 {
		// Minor version candidate is 1-2 digits (not a date which is 8 digits).
		return major + "." + parts[1]
	}

	return major
}

// isDigits reports whether s is non-empty and consists only of ASCII digits.
func isDigits(s string) bool {
	if len(s) == 0 {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
