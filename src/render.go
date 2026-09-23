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
	cAccent    = "\033[38;5;111m" // bright blue
	cMuted     = "\033[38;5;146m" // muted text
	cWhite     = "\033[38;5;254m" // white
	cOK        = "\033[38;5;114m" // green (0-50%)
	cWarn      = "\033[38;5;214m" // yellow (51-75%)
	cHigh      = "\033[38;5;208m" // orange (76-90%)
	cCrit      = "\033[38;5;196m" // red (91%+)
	cGitAdd    = "\033[38;5;114m" // green
	cGitMod    = "\033[38;5;214m" // yellow
	cGitDel    = "\033[38;5;196m" // red
	cGitAhead  = "\033[38;5;81m"  // cyan
	cGitBehind = "\033[38;5;208m" // orange
)

// paint wraps s in an ANSI color and reset.
func paint(color, s string) string {
	return color + s + cReset
}

// --- Severity levels ---

// level is a semantic severity; signals are combined with max/min.
type level int

const (
	lvlOK level = iota
	lvlWarn
	lvlHigh
	lvlCrit
)

var levelColor = [...]string{lvlOK: cOK, lvlWarn: cWarn, lvlHigh: cHigh, lvlCrit: cCrit}

// pctLevel maps a usage percentage to a severity: ≤50 ok, ≤75 warn, ≤90 high.
func pctLevel(pct int) level {
	switch {
	case pct <= 50:
		return lvlOK
	case pct <= 75:
		return lvlWarn
	case pct <= 90:
		return lvlHigh
	default:
		return lvlCrit
	}
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

func tokenLevel(tokens int) level {
	switch {
	case tokens <= ctxTokensOK:
		return lvlOK
	case tokens <= ctxTokensWarn:
		return lvlWarn
	case tokens <= ctxTokensHigh:
		return lvlHigh
	default:
		return lvlCrit
	}
}

// contextColor colors context usage by the worse of the percent-of-window band
// and the absolute-token band. Identical to the percent band for windows of
// 200k or less.
func contextColor(pct, tokens int) string {
	return levelColor[max(pctLevel(pct), tokenLevel(tokens))]
}

// Rate-limit window lengths in seconds.
const (
	fiveHourWindowSecs = 5 * 3600
	sevenDayWindowSecs = 7 * 24 * 3600
)

// Pace projection needs a meaningful slice of the window before it says
// anything: one heavy hour at the start of a 5h window (or one heavy day at
// the start of a 7d window) extrapolates to absurd numbers. Elapsed time is
// clamped to at least this fraction of the window before projecting.
const paceMinElapsedFraction = 0.25

// usageColor colors a rate-limit window by pace rather than raw fill: 80% used
// with 20 minutes left is fine, 40% used with 4.5h left will run out. It
// projects end-of-window usage from the fraction of the window elapsed.
//
// Pace can relax the raw-fill band (pctLevel) freely but raise it by at most
// one step, so a window that is barely used can never show red no matter how
// bursty the start was. Floors keep a nearly exhausted window visibly urgent
// regardless of pace. Falls back to the raw-fill bands when the reset time is
// unknown.
func usageColor(usedPct int, windowSecs int, resetsAt, now time.Time) string {
	raw := pctLevel(usedPct)
	if resetsAt.IsZero() || windowSecs <= 0 {
		return levelColor[raw]
	}
	window := float64(windowSecs)
	elapsed := window - resetsAt.Sub(now).Seconds()
	elapsed = min(max(elapsed, window*paceMinElapsedFraction), window)
	projected := float64(usedPct) * window / elapsed

	var pace level
	switch {
	case projected < 85:
		pace = lvlOK
	case projected < 100:
		pace = lvlWarn
	case projected < 120:
		pace = lvlHigh
	default:
		pace = lvlCrit
	}

	lvl := min(pace, raw+1) // pace may escalate the raw-fill band by at most one step
	// Floors: a nearly empty window blocks you no matter the pace.
	switch {
	case usedPct >= 90:
		lvl = lvlCrit
	case usedPct >= 75:
		lvl = max(lvl, lvlHigh)
	}
	return levelColor[lvl]
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

	filled := min(max(pct*barWidth/100, 0), barWidth)
	markerPos := min(max(compactThresholdPct*barWidth/100, 0), barWidth-1)

	var b strings.Builder
	for i := 0; i < barWidth; i++ {
		switch {
		case i < filled:
			b.WriteString(paint(color, "▓"))
		case compactEnabled && i == markerPos:
			b.WriteString(paint(cWarn, "▒"))
		default:
			b.WriteString(paint(cMuted, "░"))
		}
	}
	return b.String()
}

// --- Separator ---

func sep() string {
	return " " + paint(cMuted, "│") + " "
}

// --- Render ---

// Render writes two lines of ANSI-formatted statusline output to w.
func Render(w io.Writer, stdin *StdinData, git *GitStatus, compact CompactInfo) {
	fmt.Fprintf(w, "%s\n%s", buildRow1(stdin, git), buildRow2(stdin, compact, time.Now()))
}

// buildRow1 constructs: {model}[⚡][∅][•{effort}] [{agent}] │ {dir} │ {branch} [wt:{worktree}] {git_status} [#PR {state}] │ {+N/-M}
func buildRow1(stdin *StdinData, git *GitStatus) string {
	var parts []string

	// Model + fast + thinking-off + effort + agent
	modelPart := paint(cWhite, ModelDisplayName(stdin.Model.ID, stdin.Model.DisplayName))
	if stdin.FastMode {
		modelPart += paint(cWarn, "⚡")
	}
	// Thinking is on by default (and can't be disabled on some models), so
	// only the exception is marked.
	if stdin.Thinking != nil && !stdin.Thinking.Enabled {
		modelPart += paint(cMuted, "∅")
	}
	if stdin.Effort != nil && stdin.Effort.Level != "" {
		modelPart += paint(cMuted, "•") + paint(effortColor(stdin.Effort.Level), stdin.Effort.Level)
	}
	if stdin.Agent.Name != "" {
		modelPart += " " + paint(cMuted, "["+stdin.Agent.Name+"]")
	}
	parts = append(parts, modelPart)

	// Directory
	if dir := AbbreviatePath(stdin.CWD, stdin.Workspace.ProjectDir, os.Getenv("HOME")); dir != "" {
		parts = append(parts, paint(cAccent, dir))
	}

	// Git
	if git != nil && git.Branch != "" {
		gitPart := paint(cAccent, git.Branch)

		worktreeName := stdin.Workspace.GitWorktree
		if stdin.Worktree != nil && stdin.Worktree.Name != "" {
			worktreeName = stdin.Worktree.Name
		}
		if worktreeName != "" {
			gitPart += " " + paint(cMuted, "[wt:"+worktreeName+"]")
		}

		for _, st := range []struct {
			n     int
			glyph string
			color string
		}{
			{git.Added, "✚", cGitAdd},
			{git.Modified, "●", cGitMod},
			{git.Deleted, "✖", cGitDel},
			{git.Ahead, "↑", cGitAhead},
			{git.Behind, "↓", cGitBehind},
		} {
			if st.n > 0 {
				gitPart += " " + paint(st.color, fmt.Sprintf("%s%d", st.glyph, st.n))
			}
		}

		if badge := prBadge(stdin.PR); badge != "" {
			gitPart += " " + badge
		}

		parts = append(parts, gitPart)
	}

	// Lines changed
	var lines []string
	if n := stdin.Cost.TotalLinesAdded; n > 0 {
		lines = append(lines, paint(cGitAdd, fmt.Sprintf("+%d", n)))
	}
	if n := stdin.Cost.TotalLinesRemoved; n > 0 {
		lines = append(lines, paint(cGitDel, fmt.Sprintf("-%d", n)))
	}
	if len(lines) > 0 {
		parts = append(parts, strings.Join(lines, "/"))
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
	badge := paint(cMuted, fmt.Sprintf("%s%d", prefix, pr.Number))
	switch pr.ReviewState {
	case "approved":
		badge += " " + paint(cGitAdd, "✓")
	case "pending":
		badge += " " + paint(cGitMod, "⏳")
	case "changes_requested":
		badge += " " + paint(cGitDel, "✗")
	case "draft":
		badge += " " + paint(cMuted, "◌")
	}
	return badge
}

// buildRow2 constructs: {bar} {tokens}/{max} [(↻X%)] │ 5h:X% (Ym) │ 7d:X% (Ym) │ $X.XX │ Xm
func buildRow2(stdin *StdinData, compact CompactInfo, now time.Time) string {
	var fiveHour, sevenDay *RateLimitWindow
	if rl := stdin.RateLimits; rl != nil {
		fiveHour, sevenDay = rl.FiveHour, rl.SevenDay
	}
	parts := []string{
		buildContextSection(stdin.ContextWindow, compact),
		formatUsageWindow("5h", fiveHourWindowSecs, fiveHour, now),
		formatUsageWindow("7d", sevenDayWindowSecs, sevenDay, now),
	}
	if cost := FormatCost(stdin.Cost.TotalCostUSD); cost != "" {
		parts = append(parts, paint(cWhite, cost))
	}
	parts = append(parts, paint(cMuted, FormatDuration(stdin.Cost.TotalDurationMs/1000)))
	return strings.Join(parts, sep())
}

// buildContextSection builds the context bar + tokens + auto-compact marker.
// used_percentage is the source of truth for the bar; it and the token count
// are null/zero together until the first API response.
func buildContextSection(cw ContextInfo, compact CompactInfo) string {
	var pct int
	var out string
	if cw.UsedPercentage == nil {
		out = buildProgressBar(0, cOK, compact.Enabled, compact.ThresholdPct) + " " + paint(cMuted, "\u2014")
	} else {
		pct = int(*cw.UsedPercentage)
		color := contextColor(pct, cw.TotalInputTokens)
		out = buildProgressBar(pct, color, compact.Enabled, compact.ThresholdPct) + " " +
			paint(color, FormatTokens(cw.TotalInputTokens)) + "/" + paint(cMuted, FormatTokens(cw.ContextWindowSize))
	}

	if compact.Enabled {
		if pct >= compact.ThresholdPct {
			out += " " + paint(cWarn, fmt.Sprintf("(↻%d%%!)", compact.ThresholdPct))
		} else {
			out += " " + paint(cMuted, fmt.Sprintf("(↻%d%%)", compact.ThresholdPct))
		}
	}
	return out
}

// formatUsageWindow formats a single usage window like "5h:42% (2h)", colored
// by pace (see usageColor). Absent windows render as a dash.
func formatUsageWindow(label string, windowSecs int, w *RateLimitWindow, now time.Time) string {
	if w == nil {
		return paint(cMuted, label+":\u2014")
	}

	var pct int
	if w.UsedPercentage != nil {
		pct = int(*w.UsedPercentage)
	}
	var resetsAt time.Time
	if w.ResetsAt != nil {
		resetsAt = time.Unix(int64(*w.ResetsAt), 0)
	}

	result := paint(usageColor(pct, windowSecs, resetsAt, now), fmt.Sprintf("%s:%d%%", label, pct))
	if !resetsAt.IsZero() {
		result += " " + paint(cMuted, "("+FormatResetTime(resetsAt, now)+")")
	}
	return result
}
