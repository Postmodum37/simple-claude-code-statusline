package main

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
	"time"
)

// stripANSI removes ANSI escape sequences from a string for testing.
func stripANSI(s string) string {
	re := regexp.MustCompile("\x1b\\[[0-9;]*m")
	return re.ReplaceAllString(s, "")
}

// --- buildProgressBar tests ---

func TestBuildProgressBarEmpty(t *testing.T) {
	got := stripANSI(buildProgressBar(0, false, 0))
	want := strings.Repeat("░", 20)
	if got != want {
		t.Errorf("pct=0, compact=false:\n got %q\nwant %q", got, want)
	}
}

func TestBuildProgressBarHalf(t *testing.T) {
	got := stripANSI(buildProgressBar(50, false, 0))
	want := strings.Repeat("▓", 10) + strings.Repeat("░", 10)
	if got != want {
		t.Errorf("pct=50, compact=false:\n got %q\nwant %q", got, want)
	}
}

func TestBuildProgressBarFull(t *testing.T) {
	got := stripANSI(buildProgressBar(100, false, 0))
	want := strings.Repeat("▓", 20)
	if got != want {
		t.Errorf("pct=100, compact=false:\n got %q\nwant %q", got, want)
	}
}

func TestBuildProgressBarCompactMarkerVisible(t *testing.T) {
	// pct=45, compact=true, threshold=83
	// filled = 45*20/100 = 9
	// markerPos = 83*20/100 = 16
	// 9 filled + 7 empty (positions 9-15) + marker at 16 + 3 empty (17-19)
	got := stripANSI(buildProgressBar(45, true, 83))
	want := strings.Repeat("▓", 9) + strings.Repeat("░", 7) + "│" + strings.Repeat("░", 3)
	if got != want {
		t.Errorf("pct=45, compact=true, threshold=83:\n got %q (len=%d)\nwant %q (len=%d)", got, len([]rune(got)), want, len([]rune(want)))
	}
}

func TestBuildProgressBarCompactMarkerFilledPast(t *testing.T) {
	// pct=85, compact=true, threshold=83
	// filled = 85*20/100 = 17
	// markerPos = 83*20/100 = 16
	// marker at 16 < filled 17, so marker is hidden (filled over it)
	// 17 filled + 3 empty
	got := stripANSI(buildProgressBar(85, true, 83))
	want := strings.Repeat("▓", 17) + strings.Repeat("░", 3)
	if got != want {
		t.Errorf("pct=85, compact=true, threshold=83:\n got %q\nwant %q", got, want)
	}
}

func TestBuildProgressBarCompactMarkerAtEnd(t *testing.T) {
	// pct=45, compact=true, threshold=96
	// filled = 45*20/100 = 9
	// markerPos = 96*20/100 = 19 (clamped to 19)
	// 9 filled + 10 empty (positions 9-18) + marker at 19
	got := stripANSI(buildProgressBar(45, true, 96))
	want := strings.Repeat("▓", 9) + strings.Repeat("░", 10) + "│"
	if got != want {
		t.Errorf("pct=45, compact=true, threshold=96:\n got %q (len=%d)\nwant %q (len=%d)", got, len([]rune(got)), want, len([]rune(want)))
	}
}

// --- getSemanticColor tests ---

func TestGetSemanticColor(t *testing.T) {
	tests := []struct {
		name string
		pct  int
		want string
	}{
		{"0% green", 0, "\033[38;5;114m"},
		{"50% green", 50, "\033[38;5;114m"},
		{"51% yellow", 51, "\033[38;5;214m"},
		{"75% yellow", 75, "\033[38;5;214m"},
		{"76% orange", 76, "\033[38;5;208m"},
		{"90% orange", 90, "\033[38;5;208m"},
		{"91% red", 91, "\033[38;5;196m"},
		{"100% red", 100, "\033[38;5;196m"},
	}
	for _, tt := range tests {
		got := getSemanticColor(tt.pct)
		if got != tt.want {
			t.Errorf("getSemanticColor(%d) [%s] = %q, want %q", tt.pct, tt.name, got, tt.want)
		}
	}
}

// --- Full Render tests ---

func TestRenderFullData(t *testing.T) {
	var buf bytes.Buffer
	stdin := &StdinData{
		Model:     ModelInfo{ID: "claude-opus-4-6[1m]", DisplayName: "Claude Opus 4.6"},
		CWD:       "/Users/t/Workspace/project/src",
		Workspace: WorkspaceInfo{ProjectDir: "/Users/t/Workspace/project"},
		ContextWindow: ContextInfo{
			ContextWindowSize: 200000,
			UsedPercentage:    ptrFloat64(42.0),
			CurrentUsage:      &CurrentUsage{InputTokens: 60000, OutputTokens: 20000, CacheCreationInputTokens: 2000, CacheReadInputTokens: 2000},
		},
		Cost: CostInfo{
			TotalDurationMs:   300000,
			TotalCostUSD:      1.50,
			TotalLinesAdded:   42,
			TotalLinesRemoved: 7,
		},
		Agent: AgentInfo{Name: "reviewer"},
	}
	git := &GitStatus{
		Branch:   "main",
		Added:    2,
		Modified: 1,
		Deleted:  0,
		Ahead:    3,
		Behind:   1,
	}
	now := time.Now()
	usage := &UsageData{
		FetchedAt: now.Unix(),
		FiveHour: &UsageWindow{
			Utilization: 42.0,
			ResetsAt:    now.Add(2 * time.Hour).Format(time.RFC3339),
		},
		SevenDay: &UsageWindow{
			Utilization: 15.0,
			ResetsAt:    now.Add(48 * time.Hour).Format(time.RFC3339),
		},
	}
	compact := CompactInfo{Enabled: false, ThresholdPct: 0}

	Render(&buf, stdin, git, usage, compact)
	output := stripANSI(buf.String())
	lines := strings.Split(output, "\n")

	if len(lines) < 2 {
		t.Fatalf("expected 2 lines, got %d: %q", len(lines), output)
	}

	row1 := lines[0]
	row2 := lines[1]

	// Row 1 checks
	if !strings.Contains(row1, "Opus 4.6") {
		t.Errorf("row1 missing model name: %q", row1)
	}
	if !strings.Contains(row1, "[reviewer]") {
		t.Errorf("row1 missing agent: %q", row1)
	}
	if !strings.Contains(row1, "project/src") {
		t.Errorf("row1 missing directory: %q", row1)
	}
	if !strings.Contains(row1, "main") {
		t.Errorf("row1 missing git branch: %q", row1)
	}
	if !strings.Contains(row1, "+42") {
		t.Errorf("row1 missing lines added: %q", row1)
	}
	if !strings.Contains(row1, "-7") {
		t.Errorf("row1 missing lines removed: %q", row1)
	}

	// Row 2 checks
	if !strings.Contains(row2, "84k/200k") {
		t.Errorf("row2 missing tokens: %q", row2)
	}
	if !strings.Contains(row2, "5h:42%") {
		t.Errorf("row2 missing 5h usage: %q", row2)
	}
	if !strings.Contains(row2, "7d:15%") {
		t.Errorf("row2 missing 7d usage: %q", row2)
	}
	if !strings.Contains(row2, "$1.50") {
		t.Errorf("row2 missing cost: %q", row2)
	}
	if !strings.Contains(row2, "5m") {
		t.Errorf("row2 missing duration: %q", row2)
	}
}

func TestRenderMinimalData(t *testing.T) {
	var buf bytes.Buffer
	stdin := &StdinData{}
	compact := CompactInfo{}

	// Must not panic
	Render(&buf, stdin, nil, nil, compact)

	output := buf.String()
	if output == "" {
		t.Error("Render with minimal data produced empty output")
	}
	lines := strings.Split(output, "\n")
	if len(lines) < 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}
}

func TestRenderAutoCompactBelowThreshold(t *testing.T) {
	var buf bytes.Buffer
	stdin := &StdinData{
		ContextWindow: ContextInfo{
			ContextWindowSize: 200000,
			UsedPercentage:    ptrFloat64(42.0),
		},
	}
	compact := CompactInfo{Enabled: true, ThresholdPct: 83}

	Render(&buf, stdin, nil, nil, compact)
	output := stripANSI(buf.String())

	if !strings.Contains(output, "(↻83%)") {
		t.Errorf("expected (↻83%%) below threshold, got %q", output)
	}
}

func TestRenderAutoCompactPastThreshold(t *testing.T) {
	var buf bytes.Buffer
	stdin := &StdinData{
		ContextWindow: ContextInfo{
			ContextWindowSize: 200000,
			UsedPercentage:    ptrFloat64(90.0),
		},
	}
	compact := CompactInfo{Enabled: true, ThresholdPct: 83}

	Render(&buf, stdin, nil, nil, compact)
	output := stripANSI(buf.String())

	if !strings.Contains(output, "(↻83%!)") {
		t.Errorf("expected (↻83%%!) past threshold, got %q", output)
	}
}

func TestRenderExceeds200k(t *testing.T) {
	var buf bytes.Buffer
	stdin := &StdinData{
		ExceedsTokens: true,
		ContextWindow: ContextInfo{
			ContextWindowSize: 200000,
			UsedPercentage:    ptrFloat64(50.0),
		},
	}
	compact := CompactInfo{}

	Render(&buf, stdin, nil, nil, compact)
	output := stripANSI(buf.String())

	if !strings.Contains(output, ">200k") {
		t.Errorf("expected >200k indicator, got %q", output)
	}
}

func TestRenderNoUsageData(t *testing.T) {
	var buf bytes.Buffer
	stdin := &StdinData{}
	compact := CompactInfo{}

	Render(&buf, stdin, nil, nil, compact)
	output := stripANSI(buf.String())

	if !strings.Contains(output, "5h:\u2014") {
		t.Errorf("expected 5h:— with no usage, got %q", output)
	}
	if !strings.Contains(output, "7d:\u2014") {
		t.Errorf("expected 7d:— with no usage, got %q", output)
	}
}

func TestRenderStaleUsageData(t *testing.T) {
	var buf bytes.Buffer
	stdin := &StdinData{}
	usage := &UsageData{
		FetchedAt: time.Now().Add(-120 * time.Second).Unix(),
		FiveHour: &UsageWindow{
			Utilization: 42.0,
			ResetsAt:    time.Now().Add(2 * time.Hour).Format(time.RFC3339),
		},
		SevenDay: &UsageWindow{
			Utilization: 15.0,
			ResetsAt:    time.Now().Add(48 * time.Hour).Format(time.RFC3339),
		},
	}
	compact := CompactInfo{}

	Render(&buf, stdin, nil, usage, compact)
	output := stripANSI(buf.String())

	if !strings.Contains(output, "(~2m)") {
		t.Errorf("expected (~2m) staleness indicator, got %q", output)
	}
}

func TestRenderExtraUsage(t *testing.T) {
	var buf bytes.Buffer
	stdin := &StdinData{}
	usage := &UsageData{
		FetchedAt: time.Now().Unix(),
		FiveHour: &UsageWindow{
			Utilization: 10.0,
			ResetsAt:    time.Now().Add(2 * time.Hour).Format(time.RFC3339),
		},
		SevenDay: &UsageWindow{
			Utilization: 5.0,
			ResetsAt:    time.Now().Add(48 * time.Hour).Format(time.RFC3339),
		},
		ExtraUsage: &ExtraUsage{
			IsEnabled:    true,
			Utilization:  25.0,
			UsedCredits:  5000.0,
			MonthlyLimit: 20000.0,
		},
	}
	compact := CompactInfo{}

	Render(&buf, stdin, nil, usage, compact)
	output := stripANSI(buf.String())

	if !strings.Contains(output, "Extra:25%") {
		t.Errorf("expected Extra:25%% in output, got %q", output)
	}
	if !strings.Contains(output, "$50/$200") {
		t.Errorf("expected $50/$200 in output, got %q", output)
	}
}

func TestRenderContextUsedPercentageOnly(t *testing.T) {
	var buf bytes.Buffer
	stdin := &StdinData{
		ContextWindow: ContextInfo{
			ContextWindowSize: 200000,
			UsedPercentage:    ptrFloat64(42.0),
		},
	}
	compact := CompactInfo{}

	Render(&buf, stdin, nil, nil, compact)
	output := stripANSI(buf.String())

	// Bar should be at 42%, tokens estimated from percentage
	// 42% of 200k = 84k
	if !strings.Contains(output, "84k/200k") {
		t.Errorf("expected estimated tokens 84k/200k, got %q", output)
	}
}

func TestRenderContextCurrentUsageOnly(t *testing.T) {
	var buf bytes.Buffer
	stdin := &StdinData{
		ContextWindow: ContextInfo{
			ContextWindowSize: 200000,
			CurrentUsage: &CurrentUsage{
				InputTokens:  80000,
				OutputTokens: 20000,
			},
		},
	}
	compact := CompactInfo{}

	Render(&buf, stdin, nil, nil, compact)
	output := stripANSI(buf.String())

	// 100k total, 200k window = 50%
	if !strings.Contains(output, "100k/200k") {
		t.Errorf("expected 100k/200k with current_usage only, got %q", output)
	}
}

func TestRenderContextNeither(t *testing.T) {
	var buf bytes.Buffer
	stdin := &StdinData{
		ContextWindow: ContextInfo{
			ContextWindowSize: 200000,
		},
	}
	compact := CompactInfo{}

	Render(&buf, stdin, nil, nil, compact)
	output := stripANSI(buf.String())

	if !strings.Contains(output, "\u2014") {
		t.Errorf("expected — for tokens with no context data, got %q", output)
	}
}

func TestRenderUsageResetInPast(t *testing.T) {
	var buf bytes.Buffer
	stdin := &StdinData{}
	usage := &UsageData{
		FetchedAt: time.Now().Unix(),
		FiveHour: &UsageWindow{
			Utilization: 90.0,
			ResetsAt:    time.Now().Add(-10 * time.Minute).Format(time.RFC3339),
		},
		SevenDay: &UsageWindow{
			Utilization: 50.0,
			ResetsAt:    time.Now().Add(48 * time.Hour).Format(time.RFC3339),
		},
	}
	compact := CompactInfo{}

	Render(&buf, stdin, nil, usage, compact)
	output := stripANSI(buf.String())

	if !strings.Contains(output, "(old)") {
		t.Errorf("expected (old) for past reset time, got %q", output)
	}
}

// --- helpers ---

func ptrFloat64(f float64) *float64 {
	return &f
}
