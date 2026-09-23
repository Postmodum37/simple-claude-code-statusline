package main

import (
	"reflect"
	"strings"
	"testing"
)

// fullPayload mirrors the statusline JSON emitted by Claude Code v2.1.280,
// including fields this plugin ignores.
const fullPayload = `{
	"session_id": "abc-123",
	"session_name": "refactor",
	"transcript_path": "/tmp/transcript.jsonl",
	"cwd": "/home/user/project/src",
	"model": {"id": "claude-opus-5-5[1m]", "display_name": "Opus 5.5"},
	"workspace": {
		"current_dir": "/home/user/project/src",
		"project_dir": "/home/user/project",
		"added_dirs": [],
		"git_worktree": "feature-x",
		"repo": {"host": "github.com", "owner": "o", "name": "project"}
	},
	"version": "2.1.280",
	"output_style": {"name": "default"},
	"cost": {
		"total_cost_usd": 1.23,
		"total_duration_ms": 12345,
		"total_api_duration_ms": 2345,
		"total_lines_added": 100,
		"total_lines_removed": 50
	},
	"context_window": {
		"total_input_tokens": 6500,
		"total_output_tokens": 3000,
		"context_window_size": 1000000,
		"current_usage": {"input_tokens": 5000, "output_tokens": 3000, "cache_creation_input_tokens": 1000, "cache_read_input_tokens": 500},
		"used_percentage": 1,
		"remaining_percentage": 99
	},
	"exceeds_200k_tokens": false,
	"prompt_cache": {"warm": true, "ttl": "1h", "hit_ratio": 0.9},
	"fast_mode": true,
	"effort": {"level": "xhigh"},
	"thinking": {"enabled": true},
	"rate_limits": {
		"five_hour": {"used_percentage": 25.5, "resets_at": 1711200000},
		"seven_day": {"used_percentage": 10.2, "resets_at": 1711800000}
	},
	"vim": {"mode": "INSERT"},
	"agent": {"name": "reviewer"},
	"pr": {"number": 1234, "url": "https://github.com/o/project/pull/1234", "review_state": "pending", "kind": "mr"},
	"worktree": {"name": "my-feature", "path": "/p", "branch": "b", "original_cwd": "/o", "original_branch": "main"}
}`

func TestParseStdin(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  StdinData
	}{
		{
			name:  "full v2.1.280 payload",
			input: fullPayload,
			want: StdinData{
				Model:     ModelInfo{ID: "claude-opus-5-5[1m]", DisplayName: "Opus 5.5"},
				CWD:       "/home/user/project/src",
				Workspace: WorkspaceInfo{ProjectDir: "/home/user/project", GitWorktree: "feature-x"},
				ContextWindow: ContextInfo{
					ContextWindowSize: 1000000,
					UsedPercentage:    ptrFloat64(1),
					TotalInputTokens:  6500,
				},
				Cost:     CostInfo{TotalDurationMs: 12345, TotalCostUSD: 1.23, TotalLinesAdded: 100, TotalLinesRemoved: 50},
				FastMode: true,
				Agent:    AgentInfo{Name: "reviewer"},
				RateLimits: &RateLimits{
					FiveHour: &RateLimitWindow{UsedPercentage: ptrFloat64(25.5), ResetsAt: ptrFloat64(1711200000)},
					SevenDay: &RateLimitWindow{UsedPercentage: ptrFloat64(10.2), ResetsAt: ptrFloat64(1711800000)},
				},
				Worktree: &WorktreeInfo{Name: "my-feature"},
				Effort:   &EffortInfo{Level: "xhigh"},
				Thinking: &ThinkingInfo{Enabled: true},
				PR:       &PRInfo{Number: 1234, ReviewState: "pending", Kind: "mr"},
			},
		},
		{
			name:  "empty object",
			input: `{}`,
			want:  StdinData{},
		},
		{
			name: "before first API response: null context, no rate limits",
			input: `{"model": {"id": "claude-sonnet-5"}, "context_window": {
				"total_input_tokens": 0, "context_window_size": 1000000,
				"current_usage": null, "used_percentage": null, "remaining_percentage": null},
				"thinking": {"enabled": false}}`,
			want: StdinData{
				Model:         ModelInfo{ID: "claude-sonnet-5"},
				ContextWindow: ContextInfo{ContextWindowSize: 1000000},
				Thinking:      &ThinkingInfo{Enabled: false},
			},
		},
		{
			name:  "one rate-limit window, PR without review state",
			input: `{"rate_limits": {"seven_day": {"used_percentage": 3}}, "pr": {"number": 7}}`,
			want: StdinData{
				RateLimits: &RateLimits{SevenDay: &RateLimitWindow{UsedPercentage: ptrFloat64(3)}},
				PR:         &PRInfo{Number: 7},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseStdin(strings.NewReader(tt.input))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(*got, tt.want) {
				t.Errorf("ParseStdin() =\n%+v\nwant\n%+v", *got, tt.want)
			}
		})
	}
}

func TestParseStdinInvalidJSON(t *testing.T) {
	if _, err := ParseStdin(strings.NewReader(`{invalid`)); err == nil {
		t.Error("expected error for invalid JSON")
	}
}
