package main

import (
	"encoding/json"
	"io"
)

type StdinData struct {
	Model          ModelInfo     `json:"model"`
	CWD            string        `json:"cwd"`
	Workspace      WorkspaceInfo `json:"workspace"`
	ContextWindow  ContextInfo   `json:"context_window"`
	Cost           CostInfo      `json:"cost"`
	ExceedsTokens  bool          `json:"exceeds_200k_tokens"`
	SessionID      string        `json:"session_id"`
	Agent          AgentInfo     `json:"agent"`
	TranscriptPath string        `json:"transcript_path"`
	Version        string        `json:"version"`
	RateLimits     *RateLimits   `json:"rate_limits,omitempty"`
	Worktree       *WorktreeInfo `json:"worktree,omitempty"`
}

type ModelInfo struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

type WorkspaceInfo struct {
	ProjectDir string `json:"project_dir"`
}

type ContextInfo struct {
	ContextWindowSize int           `json:"context_window_size"`
	UsedPercentage    *float64      `json:"used_percentage"`
	CurrentUsage      *CurrentUsage `json:"current_usage"`
}

type CurrentUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
}

type CostInfo struct {
	TotalDurationMs   int     `json:"total_duration_ms"`
	TotalCostUSD      float64 `json:"total_cost_usd"`
	TotalLinesAdded   int     `json:"total_lines_added"`
	TotalLinesRemoved int     `json:"total_lines_removed"`
}

type AgentInfo struct {
	Name string `json:"name"`
}

type WorktreeInfo struct {
	Name           string `json:"name"`
	Path           string `json:"path"`
	Branch         string `json:"branch"`
	OriginalCWD    string `json:"original_cwd"`
	OriginalBranch string `json:"original_branch"`
}

type RateLimits struct {
	FiveHour *RateLimitWindow `json:"five_hour"`
	SevenDay *RateLimitWindow `json:"seven_day"`
}

type RateLimitWindow struct {
	UsedPercentage *float64 `json:"used_percentage"`
	ResetsAt       *float64 `json:"resets_at"`
}

// ParseStdin reads JSON from r and unmarshals it into a StdinData struct.
func ParseStdin(r io.Reader) (*StdinData, error) {
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var data StdinData
	if err := json.Unmarshal(buf, &data); err != nil {
		return nil, err
	}

	return &data, nil
}
