package main

import (
	"encoding/json"
	"io"
)

// StdinData holds the subset of Claude Code's statusline JSON this plugin
// displays. Unknown fields are ignored by encoding/json, so only consumed
// fields are declared.
type StdinData struct {
	Model         ModelInfo     `json:"model"`
	CWD           string        `json:"cwd"`
	Workspace     WorkspaceInfo `json:"workspace"`
	ContextWindow ContextInfo   `json:"context_window"`
	Cost          CostInfo      `json:"cost"`
	FastMode      bool          `json:"fast_mode"`
	Agent         AgentInfo     `json:"agent"`
	RateLimits    *RateLimits   `json:"rate_limits,omitempty"`
	Worktree      *WorktreeInfo `json:"worktree,omitempty"`
	Effort        *EffortInfo   `json:"effort,omitempty"`
	Thinking      *ThinkingInfo `json:"thinking,omitempty"`
	PR            *PRInfo       `json:"pr,omitempty"`
}

type ModelInfo struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

type WorkspaceInfo struct {
	ProjectDir string `json:"project_dir"`
	// GitWorktree is the linked git worktree's name when cwd is inside one;
	// absent in the main working tree (since v2.1.97).
	GitWorktree string `json:"git_worktree"`
}

// ContextInfo describes the current context window. UsedPercentage is null
// until the first API response; TotalInputTokens (input + cache creation +
// cache read of the latest response) is the numerator Claude Code uses for it.
type ContextInfo struct {
	ContextWindowSize int      `json:"context_window_size"`
	UsedPercentage    *float64 `json:"used_percentage"`
	TotalInputTokens  int      `json:"total_input_tokens"`
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

// WorktreeInfo is present only in `--worktree` sessions (since v2.1.69).
type WorktreeInfo struct {
	Name string `json:"name"`
}

type EffortInfo struct {
	Level string `json:"level"`
}

// PRInfo describes the open pull request for the current branch.
// Present only while Claude Code has detected an open PR (since v2.1.145).
type PRInfo struct {
	Number      int    `json:"number"`
	ReviewState string `json:"review_state"` // approved | pending | changes_requested | draft; may be absent
	Kind        string `json:"kind"`         // "mr" for GitLab merge requests; absent for GitHub PRs
}

type ThinkingInfo struct {
	Enabled bool `json:"enabled"`
}

type RateLimits struct {
	FiveHour *RateLimitWindow `json:"five_hour"`
	SevenDay *RateLimitWindow `json:"seven_day"`
}

type RateLimitWindow struct {
	UsedPercentage *float64 `json:"used_percentage"`
	ResetsAt       *float64 `json:"resets_at"` // Unix epoch seconds
}

// ParseStdin reads JSON from r and unmarshals it into a StdinData struct.
func ParseStdin(r io.Reader) (*StdinData, error) {
	var data StdinData
	if err := json.NewDecoder(r).Decode(&data); err != nil {
		return nil, err
	}
	return &data, nil
}
