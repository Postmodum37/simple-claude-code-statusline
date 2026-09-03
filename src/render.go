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

// severity ranks the semantic colors so two signals can be combined by
// taking the worse one.
func severity(color string) int {
	switch color {
	case cCrit:
		return 3
	case cHigh:
		return 2
	case cWarn:
		return 1
	default:
		return 0
	}
}

func worseColor(a, b string) string {
	if severity(b) > severity(a) {
		return b
	}
	return a
}

// Absolute-token quality bands. Model quality degrades with the number of
// tokens in context regardless of how big the window is: practitioners put
// the safe zone at roughly 100–150k tokens and the Claude Code team describes
// context rot setting in around 300–400k on 1M windows. On a 200k window these
// bands never override the percent bands; on larger windows they tighten them.
const (
	ctxTokensOK   = 150000 // ≤150k: fine
	ctxTokensWarn = 300000 // ≤300k: getting long
	ctxTokensHigh = 400000 // ≤400k: rot zone approaching; beyond: expect degraded recall
)

func absoluteTokenColor(tokens int) string {
	switch {
	case tokens <= ctxTokensOK:
		return cOK
	case tokens <= ctxTokensWarn:
		return cWarn
	case tokens <= ctxTokensHigh:
		return cHigh
	default:
		return cCrit
	}
}

// contextColor colors context usage by the worse of the percent-of-window band
// and the absolute-token band. Identical to getSemanticColor for windows of
// 200k or less.
func contextColor(pct, tokens int) string {
	return worseColor(getSemanticColor(pct), absoluteTokenColor(tokens))
}

// Rate-limit window lengths in seconds.
const (
	fiveHourWindowSecs = 5 * 3600
	sevenDayWindowSecs = 7 * 24 * 3600
)

// usageColor colors a rate-limit window by pace rather than raw fill: 80% used
// with 20 minutes left is fine, 40% used with 4.5h left will run out. It
// projects end-of-window usage from the fraction of the window elapsed.
// Floors keep a nearly exhausted window visibly urgent regardless of pace, and
// a cap keeps early-window noise from screaming while usage is still low.
// Falls back to the raw-fill bands when the reset time is unknown.
func usageColor(usedPct int, windowSecs int, resetsAt, now time.Time) string {
	if resetsAt.IsZero() || windowSecs <= 0 {
		return getSemanticColor(usedPct)
	}
	remaining := resetsAt.Sub(now).Seconds()
	elapsed := float64(windowSecs) - remaining
	minElapsed := float64(windowSecs) * 0.05
	if elapsed < minElapsed {
		elapsed = minElapsed
	}
	if elapsed > float64(windowSecs) {
		elapsed = float64(windowSecs)
	}
	projected := float64(usedPct) * float64(windowSecs) / elapsed

	var color string
	switch {
	case projected < 85:
		color = cOK
	case projected < 100:
		color = cWarn
	case projected < 120:
		color = cHigh
	default:
		color = cCrit
	}

	// Cap: with under 20% used there is plenty of headroom whatever the pace.
	if usedPct < 20 && severity(color) > severity(cWarn) {
		color = cWarn
	}
	// Floors: a nearly empty window blocks you no matter the pace.
	if usedPct >= 90 {
		color = cCrit
	} else if usedPct >= 75 {
		color = worseColor(color, cHigh)
	}
	return color
}

// effortColor returns the color for an effort level. "auto" is a mode that
// dynamically selects effort, so it gets the accent color rather than a slot
// on the cost gradient. Unknown levels fall back to muted for forward
// compatibility with future tiers.
func effortColor(level string) string {
	switch level {
	case "low":
		return cMuted
	case "medium":
		return cWhite
	case "high":
		return cWarn
	case "xhigh":
		return cHigh
	case "max":
		return cCrit
	case "auto":
		return cAccent
	default:
		return cMuted
	}
}

// --- Progress bar ---

// buildProgressBar builds a 20-character progress bar in the given color with
// an optional compact marker.
func buildProgressBar(pct int, color string, compactEnabled bool, compactThresholdPct int) string {
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

// buildRow1 constructs: {model}[⚡][*][•{effort}] [{agent}] │ {dir} │ {branch} [wt:{worktree}] {git_status} [#PR {state}] │ {+N/-M}
func buildRow1(stdin *StdinData, git *GitStatus) string {
	var parts []string

	// Model + thinking + effort + agent
	modelPart := cWhite + ModelDisplayName(stdin.Model.ID, stdin.Model.DisplayName) + cReset
	if stdin.FastMode {
		modelPart += cWarn + "⚡" + cReset
	}
	if stdin.Thinking != nil && stdin.Thinking.Enabled {
		modelPart += cMuted + "*" + cReset
	}
	if stdin.Effort != nil && stdin.Effort.Level != "" {
		modelPart += cMuted + "•" + cReset + effortColor(stdin.Effort.Level) + stdin.Effort.Level + cReset
	}
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

		worktreeName := git.Worktree
		if stdin.Worktree != nil && stdin.Worktree.Name != "" {
			worktreeName = stdin.Worktree.Name
		}
		if worktreeName != "" {
			gitPart += " " + cMuted + "[wt:" + worktreeName + "]" + cReset
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

		if badge := prBadge(stdin.PR); badge != "" {
			gitPart += " " + badge
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

// prBadge formats the open-PR badge, e.g. "#1234 ⏳" (or "!1234 ⏳" for a
// GitLab merge request, mirroring Claude Code's own footer). Returns "" when no PR.
func prBadge(pr *PRInfo) string {
	if pr == nil || pr.Number <= 0 {
		return ""
	}
	prefix := "#"
	if pr.Kind == "mr" {
		prefix = "!"
	}
	badge := cMuted + prefix + fmt.Sprintf("%d", pr.Number) + cReset
	switch pr.ReviewState {
	case "approved":
		badge += " " + cGitAdd + "✓" + cReset
	case "pending":
		badge += " " + cGitMod + "⏳" + cReset
	case "changes_requested":
		badge += " " + cGitDel + "✗" + cReset
	case "draft":
		badge += " " + cMuted + "◌" + cReset
	}
	return badge
}

// buildRow2 constructs: {bar} {tokens}/{max} [>200k] [(↻X%)] │ 5h:X% (Ym) │ 7d:X% (Ym) │ $X.XX │ Xm
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

// contextTokens returns the number of tokens occupying the context window,
// using the same definition Claude Code uses for used_percentage and
// total_input_tokens: input + cache creation + cache read. Output tokens of
// the last response are excluded so the count agrees with the percentage.
func contextTokens(u *CurrentUsage) int {
	return u.InputTokens + u.CacheCreationInputTokens + u.CacheReadInputTokens
}

// buildContextSection builds the context bar + tokens + indicators.
func buildContextSection(stdin *StdinData, compact CompactInfo) string {
	var pct int
	var tokens int
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
			tokens = contextTokens(stdin.ContextWindow.CurrentUsage)
		} else {
			// Estimate tokens from percentage
			tokens = pct * windowSize / 100
		}
		tokensStr = FormatTokens(tokens)
	} else if stdin.ContextWindow.CurrentUsage != nil {
		// Priority 2: calculate from current_usage
		tokens = contextTokens(stdin.ContextWindow.CurrentUsage)
		hasData = true
		tokensStr = FormatTokens(tokens)

		if windowSize > 0 {
			pct = tokens * 100 / windowSize
		}
	}

	var b strings.Builder

	if hasData {
		color := contextColor(pct, tokens)

		// Progress bar
		b.WriteString(buildProgressBar(pct, color, compact.Enabled, compact.ThresholdPct))
		b.WriteString(" ")

		// Tokens display
		b.WriteString(color)
		b.WriteString(tokensStr)
		b.WriteString(cReset)
		b.WriteString("/")
		b.WriteString(cMuted)
		b.WriteString(windowStr)
		b.WriteString(cReset)
	} else {
		// No data — show dash
		b.WriteString(buildProgressBar(0, cOK, compact.Enabled, compact.ThresholdPct))
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

// buildUsageSection builds the 5h/7d usage display.
func buildUsageSection(usage *UsageData) string {
	if usage == nil {
		return cMuted + "5h:\u2014" + cReset + sep() + cMuted + "7d:\u2014" + cReset
	}

	now := time.Now()
	var parts []string
	parts = append(parts, formatUsageWindow("5h", fiveHourWindowSecs, usage.FiveHour, now))
	parts = append(parts, formatUsageWindow("7d", sevenDayWindowSecs, usage.SevenDay, now))

	return strings.Join(parts, sep())
}

// formatUsageWindow formats a single usage window like "5h:42% (2h)", colored
// by pace (see usageColor).
func formatUsageWindow(label string, windowSecs int, window *UsageWindow, now time.Time) string {
	if window == nil {
		return cMuted + label + ":\u2014" + cReset
	}

	pct := int(window.Utilization)

	var resetsAt time.Time
	if window.ResetsAt != "" {
		if t, err := time.Parse(time.RFC3339, window.ResetsAt); err == nil {
			resetsAt = t
		}
	}
	color := usageColor(pct, windowSecs, resetsAt, now)

	result := color + fmt.Sprintf("%s:%d%%", label, pct) + cReset

	if !resetsAt.IsZero() {
		resetStr := FormatResetTime(window.ResetsAt, now)
		result += " " + cMuted + "(" + resetStr + ")" + cReset
	}

	return result
}
