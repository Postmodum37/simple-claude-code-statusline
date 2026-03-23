package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// --- Color constants (Tokyo Night palette) ---

const (
	cReset     = "\033[0m"
	cAccent    = "\033[38;5;111m"  // bright blue
	cMuted     = "\033[38;5;146m"  // muted text
	cWhite     = "\033[38;5;254m"  // white
	cOK        = "\033[38;5;114m"  // green (0-50%)
	cWarn      = "\033[38;5;214m"  // yellow (51-75%)
	cHigh      = "\033[38;5;208m"  // orange (76-90%)
	cCrit      = "\033[38;5;196m"  // red (91%+)
	cGitAdd    = "\033[38;5;114m"  // green
	cGitMod    = "\033[38;5;214m"  // yellow
	cGitDel    = "\033[38;5;196m"  // red
	cGitAhead  = "\033[38;5;81m"   // cyan
	cGitBehind = "\033[38;5;208m"  // orange
)

// CompactInfo holds auto-compact state for the progress bar.
type CompactInfo struct {
	Enabled      bool
	ThresholdPct int
}

// --- Semantic color ---

// getSemanticColor returns an ANSI color code based on a usage percentage.
func getSemanticColor(pct int) string {
	switch {
	case pct <= 50:
		return cOK
	case pct <= 75:
		return cWarn
	case pct <= 90:
		return cHigh
	default:
		return cCrit
	}
}

// --- Progress bar ---

// buildProgressBar builds a 20-character progress bar with optional compact marker.
func buildProgressBar(pct int, compactEnabled bool, compactThresholdPct int) string {
	const barWidth = 20

	filled := pct * barWidth / 100
	if filled < 0 {
		filled = 0
	}
	if filled > barWidth {
		filled = barWidth
	}

	markerPos := compactThresholdPct * barWidth / 100
	if markerPos < 0 {
		markerPos = 0
	}
	if markerPos > barWidth-1 {
		markerPos = barWidth - 1
	}

	color := getSemanticColor(pct)

	var b strings.Builder
	for i := 0; i < barWidth; i++ {
		if i < filled {
			b.WriteString(color)
			b.WriteString("▓")
			b.WriteString(cReset)
		} else if compactEnabled && i == markerPos && i >= filled {
			b.WriteString(cWarn)
			b.WriteString("▒")
			b.WriteString(cReset)
		} else {
			b.WriteString(cMuted)
			b.WriteString("░")
			b.WriteString(cReset)
		}
	}
	return b.String()
}

// --- Separator ---

func sep() string {
	return " " + cMuted + "│" + cReset + " "
}

// --- Render ---

// Render writes two lines of ANSI-formatted statusline output to w.
func Render(w io.Writer, stdin *StdinData, git *GitStatus, usage *UsageData, compact CompactInfo) {
	row1 := buildRow1(stdin, git)
	row2 := buildRow2(stdin, usage, compact)
	fmt.Fprintf(w, "%s\n%s", row1, row2)
}

// buildRow1 constructs: {model} [{agent}] │ {dir} │ {branch} [wt:{worktree}] {git_status} │ {+N/-M}
func buildRow1(stdin *StdinData, git *GitStatus) string {
	var parts []string

	// Model + agent
	modelPart := cWhite + ModelDisplayName(stdin.Model.ID, stdin.Model.DisplayName) + cReset
	if stdin.Agent.Name != "" {
		modelPart += " " + cMuted + "[" + stdin.Agent.Name + "]" + cReset
	}
	parts = append(parts, modelPart)

	// Directory
	home := os.Getenv("HOME")
	dir := AbbreviatePath(stdin.CWD, stdin.Workspace.ProjectDir, home)
	if dir != "" {
		parts = append(parts, cAccent+dir+cReset)
	}

	// Git
	if git != nil && git.Branch != "" {
		gitPart := cAccent + git.Branch + cReset

		if git.Worktree != "" {
			gitPart += " " + cMuted + "[wt:" + git.Worktree + "]" + cReset
		}

		var statusParts []string
		if git.Added > 0 {
			statusParts = append(statusParts, fmt.Sprintf("%s✚%d%s", cGitAdd, git.Added, cReset))
		}
		if git.Modified > 0 {
			statusParts = append(statusParts, fmt.Sprintf("%s●%d%s", cGitMod, git.Modified, cReset))
		}
		if git.Deleted > 0 {
			statusParts = append(statusParts, fmt.Sprintf("%s✖%d%s", cGitDel, git.Deleted, cReset))
		}
		if git.Ahead > 0 {
			statusParts = append(statusParts, fmt.Sprintf("%s↑%d%s", cGitAhead, git.Ahead, cReset))
		}
		if git.Behind > 0 {
			statusParts = append(statusParts, fmt.Sprintf("%s↓%d%s", cGitBehind, git.Behind, cReset))
		}
		if len(statusParts) > 0 {
			gitPart += " " + strings.Join(statusParts, " ")
		}

		parts = append(parts, gitPart)
	}

	// Lines changed
	if stdin.Cost.TotalLinesAdded > 0 || stdin.Cost.TotalLinesRemoved > 0 {
		var linesParts []string
		if stdin.Cost.TotalLinesAdded > 0 {
			linesParts = append(linesParts, fmt.Sprintf("%s+%d%s", cGitAdd, stdin.Cost.TotalLinesAdded, cReset))
		}
		if stdin.Cost.TotalLinesRemoved > 0 {
			linesParts = append(linesParts, fmt.Sprintf("%s-%d%s", cGitDel, stdin.Cost.TotalLinesRemoved, cReset))
		}
		parts = append(parts, strings.Join(linesParts, "/"))
	}

	return strings.Join(parts, sep())
}

// buildRow2 constructs: {bar} {tokens}/{max} [>200k] [(↻X%)] │ 5h:X% (Ym) │ 7d:X% (Ym) │ [Extra:X%] │ $X.XX │ Xm
func buildRow2(stdin *StdinData, usage *UsageData, compact CompactInfo) string {
	var parts []string

	// --- Context section ---
	contextPart := buildContextSection(stdin, compact)
	parts = append(parts, contextPart)

	// --- Usage section ---
	usagePart := buildUsageSection(usage)
	parts = append(parts, usagePart)

	// --- Cost ---
	costStr := FormatCost(stdin.Cost.TotalCostUSD)
	if costStr != "" {
		parts = append(parts, cWhite+costStr+cReset)
	}

	// --- Duration ---
	durationSecs := stdin.Cost.TotalDurationMs / 1000
	parts = append(parts, cMuted+FormatDuration(durationSecs)+cReset)

	return strings.Join(parts, sep())
}

// buildContextSection builds the context bar + tokens + indicators.
func buildContextSection(stdin *StdinData, compact CompactInfo) string {
	var pct int
	var tokensStr string
	hasData := false

	windowSize := stdin.ContextWindow.ContextWindowSize
	windowStr := FormatTokens(windowSize)

	if stdin.ContextWindow.UsedPercentage != nil {
		// Priority 1: used_percentage is the source of truth for bar/color
		pct = int(*stdin.ContextWindow.UsedPercentage)
		hasData = true

		if stdin.ContextWindow.CurrentUsage != nil {
			// Use actual token counts for display
			total := stdin.ContextWindow.CurrentUsage.InputTokens +
				stdin.ContextWindow.CurrentUsage.OutputTokens +
				stdin.ContextWindow.CurrentUsage.CacheCreationInputTokens +
				stdin.ContextWindow.CurrentUsage.CacheReadInputTokens
			tokensStr = FormatTokens(total)
		} else {
			// Estimate tokens from percentage
			estimated := pct * windowSize / 100
			tokensStr = FormatTokens(estimated)
		}
	} else if stdin.ContextWindow.CurrentUsage != nil {
		// Priority 2: calculate from current_usage
		total := stdin.ContextWindow.CurrentUsage.InputTokens +
			stdin.ContextWindow.CurrentUsage.OutputTokens +
			stdin.ContextWindow.CurrentUsage.CacheCreationInputTokens +
			stdin.ContextWindow.CurrentUsage.CacheReadInputTokens
		hasData = true
		tokensStr = FormatTokens(total)

		if windowSize > 0 {
			pct = total * 100 / windowSize
		}
	}

	var b strings.Builder

	if hasData {
		// Progress bar
		b.WriteString(buildProgressBar(pct, compact.Enabled, compact.ThresholdPct))
		b.WriteString(" ")

		// Tokens display
		color := getSemanticColor(pct)
		b.WriteString(color)
		b.WriteString(tokensStr)
		b.WriteString(cReset)
		b.WriteString("/")
		b.WriteString(cMuted)
		b.WriteString(windowStr)
		b.WriteString(cReset)
	} else {
		// No data — show dash
		b.WriteString(buildProgressBar(0, compact.Enabled, compact.ThresholdPct))
		b.WriteString(" ")
		b.WriteString(cMuted)
		b.WriteString("\u2014") // em dash
		b.WriteString(cReset)
	}

	// >200k indicator
	if stdin.ExceedsTokens {
		b.WriteString(" ")
		b.WriteString(cHigh)
		b.WriteString(">200k")
		b.WriteString(cReset)
	}

	// Auto-compact indicator
	if compact.Enabled {
		b.WriteString(" ")
		if pct >= compact.ThresholdPct {
			b.WriteString(cWarn)
			b.WriteString(fmt.Sprintf("(↻%d%%!)", compact.ThresholdPct))
			b.WriteString(cReset)
		} else {
			b.WriteString(cMuted)
			b.WriteString(fmt.Sprintf("(↻%d%%)", compact.ThresholdPct))
			b.WriteString(cReset)
		}
	}

	return b.String()
}

// buildUsageSection builds the 5h/7d/extra usage display.
func buildUsageSection(usage *UsageData) string {
	if usage == nil {
		return cMuted + "5h:\u2014" + cReset + sep() + cMuted + "7d:\u2014" + cReset
	}

	var parts []string

	// 5h window
	parts = append(parts, formatUsageWindow("5h", usage.FiveHour))

	// 7d window
	parts = append(parts, formatUsageWindow("7d", usage.SevenDay))

	// Check for any reset time in the past
	now := time.Now()
	hasOldReset := false
	if usage.FiveHour != nil && usage.FiveHour.ResetsAt != "" {
		if resetTime, err := time.Parse(time.RFC3339, usage.FiveHour.ResetsAt); err == nil {
			if resetTime.Before(now) {
				hasOldReset = true
			}
		}
	}
	if usage.SevenDay != nil && usage.SevenDay.ResetsAt != "" {
		if resetTime, err := time.Parse(time.RFC3339, usage.SevenDay.ResetsAt); err == nil {
			if resetTime.Before(now) {
				hasOldReset = true
			}
		}
	}

	// Staleness indicator
	age := now.Unix() - usage.FetchedAt
	if age > 60 {
		mins := age / 60
		parts = append(parts, cMuted+fmt.Sprintf("(~%dm)", mins)+cReset)
	}

	if hasOldReset {
		parts = append(parts, cWarn+"(old)"+cReset)
	}

	// Extra usage
	if usage.ExtraUsage != nil && usage.ExtraUsage.IsEnabled {
		pct := int(usage.ExtraUsage.Utilization)
		color := getSemanticColor(pct)
		extraStr := fmt.Sprintf("%sExtra:%d%%%s ($%d/$%d)",
			color, pct, cReset,
			int(usage.ExtraUsage.UsedCredits/100),
			int(usage.ExtraUsage.MonthlyLimit/100))
		parts = append(parts, extraStr)
	}

	return strings.Join(parts, sep())
}

// formatUsageWindow formats a single usage window like "5h:42% (2h)".
func formatUsageWindow(label string, window *UsageWindow) string {
	if window == nil {
		return cMuted + label + ":\u2014" + cReset
	}

	pct := int(window.Utilization)
	color := getSemanticColor(pct)

	result := color + fmt.Sprintf("%s:%d%%", label, pct) + cReset

	if window.ResetsAt != "" {
		resetStr := FormatResetTime(window.ResetsAt, time.Now())
		result += " " + cMuted + "(" + resetStr + ")" + cReset
	}

	return result
}
