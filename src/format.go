package main

import (
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"time"
)

// FormatTokens formats a token count with human-readable suffixes.
// e.g., 42000 → "42k", 1000000 → "1m"
func FormatTokens(n int) string {
	if n >= 1000000 {
		return fmt.Sprintf("%dm", n/1000000)
	}
	if n >= 1000 {
		return fmt.Sprintf("%dk", n/1000)
	}
	return fmt.Sprintf("%d", n)
}

// FormatDuration formats seconds into a human-readable duration.
// Days: "1d0h", Hours: "1h5m", Minutes: "3m". Negative values clamped to 0.
func FormatDuration(secs int) string {
	if secs < 0 {
		secs = 0
	}
	mins := secs / 60
	hours := mins / 60
	days := hours / 24

	if days > 0 {
		return fmt.Sprintf("%dd%dh", days, hours%24)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh%dm", hours, mins%60)
	}
	return fmt.Sprintf("%dm", mins)
}

// FormatCost formats a USD cost for display.
// Returns empty string for zero. Uses 2 decimal places under $10, integer above.
func FormatCost(usd float64) string {
	if usd == 0 {
		return ""
	}
	intPart := int(math.Floor(usd))
	if intPart >= 10 {
		return fmt.Sprintf("$%d", intPart)
	}
	return fmt.Sprintf("$%.2f", usd)
}

// FormatResetTime formats an ISO 8601 reset timestamp relative to now.
// Skips zero sub-units (e.g., 2h exactly → "2h", not "2h0m").
// Returns "0m" for empty string or past timestamps.
func FormatResetTime(resetISO string, now time.Time) string {
	if resetISO == "" {
		return "0m"
	}

	resetTime, err := time.Parse(time.RFC3339, resetISO)
	if err != nil {
		return "0m"
	}

	diff := int(resetTime.Sub(now).Seconds())
	if diff < 0 {
		return "0m"
	}

	days := diff / 86400
	hours := (diff % 86400) / 3600
	mins := (diff % 3600) / 60

	if days > 0 {
		if hours > 0 {
			return fmt.Sprintf("%dd%dh", days, hours)
		}
		return fmt.Sprintf("%dd", days)
	}
	if hours > 0 {
		if mins > 0 {
			return fmt.Sprintf("%dh%dm", hours, mins)
		}
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dm", mins)
}

// AbbreviatePath abbreviates a directory path for display.
// If inside a project (projectDir non-empty and cwd starts with it), shows "repoName/relative".
// Otherwise, uses fish-style abbreviation: intermediate segments shortened to first char
// (hidden dirs keep dot+first char), last segment kept full.
func AbbreviatePath(cwd, projectDir, home string) string {
	// If inside a git project, show repo-relative path
	if projectDir != "" && strings.HasPrefix(cwd, projectDir) {
		repoName := filepath.Base(projectDir)
		if cwd == projectDir {
			return repoName
		}
		relative := cwd[len(projectDir)+1:]
		return repoName + "/" + relative
	}

	// Replace home prefix with ~
	display := cwd
	if home != "" && strings.HasPrefix(cwd, home) {
		display = "~" + cwd[len(home):]
	}

	// Split into parts
	parts := strings.Split(display, "/")

	// Filter out empty parts but track if path starts with /
	var segments []string
	startsWithSlash := strings.HasPrefix(display, "/")
	for _, p := range parts {
		if p != "" {
			segments = append(segments, p)
		}
	}

	// Short path (2 or fewer segments): show as-is
	// For absolute paths: /tmp has segments ["tmp"] → 1 segment
	// For ~/foo: segments ["~", "foo"] → 2 segments
	totalParts := len(segments)
	if startsWithSlash {
		totalParts++ // count the leading / as a "part"
	}
	if totalParts <= 2 {
		return display
	}

	// Abbreviate all but the last segment
	var abbreviated strings.Builder
	if startsWithSlash && (len(segments) == 0 || segments[0] != "~") {
		abbreviated.WriteString("/")
	}
	for i := 0; i < len(segments)-1; i++ {
		seg := segments[i]
		if seg == "~" {
			abbreviated.WriteString("~")
		} else if strings.HasPrefix(seg, ".") {
			// Hidden dir: dot + first char
			if len(seg) >= 2 {
				abbreviated.WriteString(seg[:2])
			} else {
				abbreviated.WriteString(seg)
			}
		} else {
			abbreviated.WriteString(seg[:1])
		}
		abbreviated.WriteString("/")
	}
	// Add full last segment
	abbreviated.WriteString(segments[len(segments)-1])
	return abbreviated.String()
}
