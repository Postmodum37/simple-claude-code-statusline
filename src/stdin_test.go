package main

import (
	"strings"
	"testing"
)

func TestParseStdin_FullValidJSON(t *testing.T) {
	input := `{
		"model": {"id": "claude-opus-4-6", "display_name": "Claude Opus 4.6"},
		"cwd": "/home/user/project",
		"workspace": {"project_dir": "/home/user/project"},
		"context_window": {
			"context_window_size": 200000,
			"used_percentage": 42.7,
			"current_usage": {
				"input_tokens": 5000,
				"output_tokens": 3000,
				"cache_creation_input_tokens": 1000,
				"cache_read_input_tokens": 500
			}
		},
		"cost": {
			"total_duration_ms": 12345,
			"total_cost_usd": 1.23,
			"total_lines_added": 100,
			"total_lines_removed": 50
		},
		"exceeds_200k_tokens": true,
		"session_id": "abc-123",
		"agent": {"name": "myagent"},
		"transcript_path": "/tmp/transcript.json",
		"version": "2.1.63",
		"rate_limits": {
			"five_hour": {"used_percentage": 25.5, "resets_at": 1711200000.0},
			"seven_day": {"used_percentage": 10.2, "resets_at": 1711800000.0}
		}
	}`

	data, err := ParseStdin(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Model
	if data.Model.ID != "claude-opus-4-6" {
		t.Errorf("Model.ID = %q, want %q", data.Model.ID, "claude-opus-4-6")
	}
	if data.Model.DisplayName != "Claude Opus 4.6" {
		t.Errorf("Model.DisplayName = %q, want %q", data.Model.DisplayName, "Claude Opus 4.6")
	}

	// CWD
	if data.CWD != "/home/user/project" {
		t.Errorf("CWD = %q, want %q", data.CWD, "/home/user/project")
	}

	// Workspace
	if data.Workspace.ProjectDir != "/home/user/project" {
		t.Errorf("Workspace.ProjectDir = %q, want %q", data.Workspace.ProjectDir, "/home/user/project")
	}

	// ContextWindow
	if data.ContextWindow.ContextWindowSize != 200000 {
		t.Errorf("ContextWindow.ContextWindowSize = %d, want %d", data.ContextWindow.ContextWindowSize, 200000)
	}
	if data.ContextWindow.UsedPercentage == nil {
		t.Fatal("ContextWindow.UsedPercentage is nil, want non-nil")
	}
	if *data.ContextWindow.UsedPercentage != 42.7 {
		t.Errorf("ContextWindow.UsedPercentage = %v, want %v", *data.ContextWindow.UsedPercentage, 42.7)
	}
	if data.ContextWindow.CurrentUsage == nil {
		t.Fatal("ContextWindow.CurrentUsage is nil, want non-nil")
	}
	if data.ContextWindow.CurrentUsage.InputTokens != 5000 {
		t.Errorf("CurrentUsage.InputTokens = %d, want %d", data.ContextWindow.CurrentUsage.InputTokens, 5000)
	}
	if data.ContextWindow.CurrentUsage.OutputTokens != 3000 {
		t.Errorf("CurrentUsage.OutputTokens = %d, want %d", data.ContextWindow.CurrentUsage.OutputTokens, 3000)
	}
	if data.ContextWindow.CurrentUsage.CacheCreationInputTokens != 1000 {
		t.Errorf("CurrentUsage.CacheCreationInputTokens = %d, want %d", data.ContextWindow.CurrentUsage.CacheCreationInputTokens, 1000)
	}
	if data.ContextWindow.CurrentUsage.CacheReadInputTokens != 500 {
		t.Errorf("CurrentUsage.CacheReadInputTokens = %d, want %d", data.ContextWindow.CurrentUsage.CacheReadInputTokens, 500)
	}

	// Cost
	if data.Cost.TotalDurationMs != 12345 {
		t.Errorf("Cost.TotalDurationMs = %d, want %d", data.Cost.TotalDurationMs, 12345)
	}
	if data.Cost.TotalCostUSD != 1.23 {
		t.Errorf("Cost.TotalCostUSD = %v, want %v", data.Cost.TotalCostUSD, 1.23)
	}
	if data.Cost.TotalLinesAdded != 100 {
		t.Errorf("Cost.TotalLinesAdded = %d, want %d", data.Cost.TotalLinesAdded, 100)
	}
	if data.Cost.TotalLinesRemoved != 50 {
		t.Errorf("Cost.TotalLinesRemoved = %d, want %d", data.Cost.TotalLinesRemoved, 50)
	}

	// ExceedsTokens
	if !data.ExceedsTokens {
		t.Error("ExceedsTokens = false, want true")
	}

	// SessionID
	if data.SessionID != "abc-123" {
		t.Errorf("SessionID = %q, want %q", data.SessionID, "abc-123")
	}

	// Agent
	if data.Agent.Name != "myagent" {
		t.Errorf("Agent.Name = %q, want %q", data.Agent.Name, "myagent")
	}

	// TranscriptPath
	if data.TranscriptPath != "/tmp/transcript.json" {
		t.Errorf("TranscriptPath = %q, want %q", data.TranscriptPath, "/tmp/transcript.json")
	}

	// Version
	if data.Version != "2.1.63" {
		t.Errorf("Version = %q, want %q", data.Version, "2.1.63")
	}

	// RateLimits
	if data.RateLimits == nil {
		t.Fatal("RateLimits is nil, want non-nil")
	}
	if data.RateLimits.FiveHour == nil {
		t.Fatal("RateLimits.FiveHour is nil, want non-nil")
	}
	if data.RateLimits.FiveHour.UsedPercentage == nil || *data.RateLimits.FiveHour.UsedPercentage != 25.5 {
		t.Errorf("RateLimits.FiveHour.UsedPercentage = %v, want 25.5", data.RateLimits.FiveHour.UsedPercentage)
	}
	if data.RateLimits.FiveHour.ResetsAt == nil || *data.RateLimits.FiveHour.ResetsAt != 1711200000.0 {
		t.Errorf("RateLimits.FiveHour.ResetsAt = %v, want 1711200000.0", data.RateLimits.FiveHour.ResetsAt)
	}
	if data.RateLimits.SevenDay == nil {
		t.Fatal("RateLimits.SevenDay is nil, want non-nil")
	}
	if data.RateLimits.SevenDay.UsedPercentage == nil || *data.RateLimits.SevenDay.UsedPercentage != 10.2 {
		t.Errorf("RateLimits.SevenDay.UsedPercentage = %v, want 10.2", data.RateLimits.SevenDay.UsedPercentage)
	}
	if data.RateLimits.SevenDay.ResetsAt == nil || *data.RateLimits.SevenDay.ResetsAt != 1711800000.0 {
		t.Errorf("RateLimits.SevenDay.ResetsAt = %v, want 1711800000.0", data.RateLimits.SevenDay.ResetsAt)
	}
}

func TestParseStdin_MinimalJSON(t *testing.T) {
	data, err := ParseStdin(strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if data.Model.ID != "" {
		t.Errorf("Model.ID = %q, want empty", data.Model.ID)
	}
	if data.CWD != "" {
		t.Errorf("CWD = %q, want empty", data.CWD)
	}
	if data.ContextWindow.ContextWindowSize != 0 {
		t.Errorf("ContextWindow.ContextWindowSize = %d, want 0", data.ContextWindow.ContextWindowSize)
	}
	if data.ContextWindow.UsedPercentage != nil {
		t.Errorf("ContextWindow.UsedPercentage = %v, want nil", data.ContextWindow.UsedPercentage)
	}
	if data.ContextWindow.CurrentUsage != nil {
		t.Errorf("ContextWindow.CurrentUsage = %v, want nil", data.ContextWindow.CurrentUsage)
	}
	if data.Cost.TotalCostUSD != 0 {
		t.Errorf("Cost.TotalCostUSD = %v, want 0", data.Cost.TotalCostUSD)
	}
	if data.ExceedsTokens {
		t.Error("ExceedsTokens = true, want false")
	}
	if data.SessionID != "" {
		t.Errorf("SessionID = %q, want empty", data.SessionID)
	}
	if data.RateLimits != nil {
		t.Errorf("RateLimits = %v, want nil", data.RateLimits)
	}
}

func TestParseStdin_UsedPercentageFloat(t *testing.T) {
	input := `{"context_window": {"used_percentage": 42.7}}`
	data, err := ParseStdin(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.ContextWindow.UsedPercentage == nil {
		t.Fatal("ContextWindow.UsedPercentage is nil, want non-nil")
	}
	if *data.ContextWindow.UsedPercentage != 42.7 {
		t.Errorf("ContextWindow.UsedPercentage = %v, want 42.7", *data.ContextWindow.UsedPercentage)
	}
}

func TestParseStdin_RateLimitsPresent(t *testing.T) {
	input := `{"rate_limits": {"five_hour": {"used_percentage": 10.0}}}`
	data, err := ParseStdin(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.RateLimits == nil {
		t.Fatal("RateLimits is nil, want non-nil")
	}
	if data.RateLimits.FiveHour == nil {
		t.Fatal("RateLimits.FiveHour is nil, want non-nil")
	}
	if data.RateLimits.FiveHour.UsedPercentage == nil || *data.RateLimits.FiveHour.UsedPercentage != 10.0 {
		t.Errorf("RateLimits.FiveHour.UsedPercentage = %v, want 10.0", data.RateLimits.FiveHour.UsedPercentage)
	}
}

func TestParseStdin_RateLimitsAbsent(t *testing.T) {
	input := `{"model": {"id": "test"}}`
	data, err := ParseStdin(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.RateLimits != nil {
		t.Errorf("RateLimits = %v, want nil", data.RateLimits)
	}
}

func TestParseStdin_NullableFields(t *testing.T) {
	// extra_usage.utilization can be null in the usage API response;
	// here we test that null values in JSON don't cause errors
	// and result in zero/nil values
	input := `{"context_window": {"used_percentage": null, "current_usage": null}}`
	data, err := ParseStdin(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.ContextWindow.UsedPercentage != nil {
		t.Errorf("ContextWindow.UsedPercentage = %v, want nil", data.ContextWindow.UsedPercentage)
	}
	if data.ContextWindow.CurrentUsage != nil {
		t.Errorf("ContextWindow.CurrentUsage = %v, want nil", data.ContextWindow.CurrentUsage)
	}
}

func TestParseStdin_ModelIDWith1MSuffix(t *testing.T) {
	input := `{"model": {"id": "claude-opus-4-6[1m]"}}`
	data, err := ParseStdin(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The [1m] suffix should be preserved in the struct — stripping happens in model.go
	if data.Model.ID != "claude-opus-4-6[1m]" {
		t.Errorf("Model.ID = %q, want %q", data.Model.ID, "claude-opus-4-6[1m]")
	}
}

func TestParseStdin_ExceedsTokensTrue(t *testing.T) {
	input := `{"exceeds_200k_tokens": true}`
	data, err := ParseStdin(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !data.ExceedsTokens {
		t.Error("ExceedsTokens = false, want true")
	}
}

func TestParseStdin_ExceedsTokensFalse(t *testing.T) {
	input := `{"exceeds_200k_tokens": false}`
	data, err := ParseStdin(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.ExceedsTokens {
		t.Error("ExceedsTokens = true, want false")
	}
}

func TestParseStdin_InvalidJSON(t *testing.T) {
	_, err := ParseStdin(strings.NewReader(`{invalid`))
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}
