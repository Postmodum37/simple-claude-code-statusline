package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetCompactThreshold(t *testing.T) {
	tests := []struct {
		name              string
		contextWindowSize int
		claudeJSON        string // contents to write; empty = no file
		noFile            bool   // if true, use non-existent path
		envVars           map[string]string
		wantEnabled       bool
		wantThresholdPct  int
	}{
		{
			name:              "200k context, auto-compact enabled by default (no file)",
			contextWindowSize: 200000,
			noFile:            true,
			wantEnabled:       true,
			wantThresholdPct:  83, // (200000-20000-13000)*100/200000 = 167000*100/200000 = 83.5 → 83
		},
		{
			name:              "200k context, auto-compact enabled by default (field absent)",
			contextWindowSize: 200000,
			claudeJSON:        `{"theme": "dark"}`,
			wantEnabled:       true,
			wantThresholdPct:  83,
		},
		{
			name:              "200k context, auto-compact explicitly enabled",
			contextWindowSize: 200000,
			claudeJSON:        `{"autoCompactEnabled": true}`,
			wantEnabled:       true,
			wantThresholdPct:  83,
		},
		{
			name:              "1M context, no env vars",
			contextWindowSize: 1000000,
			noFile:            true,
			wantEnabled:       true,
			wantThresholdPct:  96, // (1000000-20000-13000)*100/1000000 = 967000*100/1000000 = 96.7 → 96
		},
		{
			name:              "200k context, CLAUDE_AUTOCOMPACT_PCT_OVERRIDE=70",
			contextWindowSize: 200000,
			noFile:            true,
			envVars:           map[string]string{"CLAUDE_AUTOCOMPACT_PCT_OVERRIDE": "70"},
			wantEnabled:       true,
			wantThresholdPct:  63, // 70% of 180000 = 126000, 126000*100/200000 = 63%
		},
		{
			name:              "200k context, CLAUDE_AUTOCOMPACT_PCT_OVERRIDE=99 (clamped to default)",
			contextWindowSize: 200000,
			noFile:            true,
			envVars:           map[string]string{"CLAUDE_AUTOCOMPACT_PCT_OVERRIDE": "99"},
			wantEnabled:       true,
			wantThresholdPct:  83, // 99% of 180000 = 178200, but default is 167000, min → 167000*100/200000 = 83
		},
		{
			name:              "200k context, CLAUDE_CODE_AUTO_COMPACT_WINDOW=100000",
			contextWindowSize: 200000,
			noFile:            true,
			envVars:           map[string]string{"CLAUDE_CODE_AUTO_COMPACT_WINDOW": "100000"},
			wantEnabled:       true,
			wantThresholdPct:  33, // cappedWindow=100000, effective=80000, threshold=67000, 67000*100/200000=33
		},
		{
			name:              "auto-compact disabled",
			contextWindowSize: 200000,
			claudeJSON:        `{"autoCompactEnabled": false}`,
			wantEnabled:       false,
			wantThresholdPct:  0,
		},
		{
			name:              "auto-compact disabled with 1M context",
			contextWindowSize: 1000000,
			claudeJSON:        `{"autoCompactEnabled": false}`,
			wantEnabled:       false,
			wantThresholdPct:  0,
		},
		{
			name:              "degenerate: context window too small",
			contextWindowSize: 20000,
			noFile:            true,
			wantEnabled:       false,
			wantThresholdPct:  0, // effectiveWindow = 0, degenerate
		},
		{
			name:              "CLAUDE_CODE_AUTO_COMPACT_WINDOW smaller than outputReserve",
			contextWindowSize: 200000,
			noFile:            true,
			envVars:           map[string]string{"CLAUDE_CODE_AUTO_COMPACT_WINDOW": "10000"},
			wantEnabled:       false,
			wantThresholdPct:  0, // effectiveWindow = -10000, degenerate
		},
		{
			name:              "invalid CLAUDE_AUTOCOMPACT_PCT_OVERRIDE ignored",
			contextWindowSize: 200000,
			noFile:            true,
			envVars:           map[string]string{"CLAUDE_AUTOCOMPACT_PCT_OVERRIDE": "abc"},
			wantEnabled:       true,
			wantThresholdPct:  83, // falls back to default
		},
		{
			name:              "CLAUDE_AUTOCOMPACT_PCT_OVERRIDE=0 ignored (out of 1-100 range)",
			contextWindowSize: 200000,
			noFile:            true,
			envVars:           map[string]string{"CLAUDE_AUTOCOMPACT_PCT_OVERRIDE": "0"},
			wantEnabled:       true,
			wantThresholdPct:  83,
		},
		{
			name:              "CLAUDE_AUTOCOMPACT_PCT_OVERRIDE=101 ignored (out of 1-100 range)",
			contextWindowSize: 200000,
			noFile:            true,
			envVars:           map[string]string{"CLAUDE_AUTOCOMPACT_PCT_OVERRIDE": "101"},
			wantEnabled:       true,
			wantThresholdPct:  83,
		},
		{
			name:              "invalid claude.json contents defaults to enabled",
			contextWindowSize: 200000,
			claudeJSON:        `not valid json`,
			wantEnabled:       true,
			wantThresholdPct:  83,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up env vars
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			// Set up claude.json path
			var claudeJSONPath string
			if tt.noFile {
				claudeJSONPath = filepath.Join(t.TempDir(), "nonexistent", "claude.json")
			} else {
				dir := t.TempDir()
				claudeJSONPath = filepath.Join(dir, "claude.json")
				if err := os.WriteFile(claudeJSONPath, []byte(tt.claudeJSON), 0644); err != nil {
					t.Fatalf("failed to write test claude.json: %v", err)
				}
			}

			gotEnabled, gotThresholdPct := GetCompactThreshold(tt.contextWindowSize, claudeJSONPath)
			if gotEnabled != tt.wantEnabled {
				t.Errorf("enabled = %v, want %v", gotEnabled, tt.wantEnabled)
			}
			if gotThresholdPct != tt.wantThresholdPct {
				t.Errorf("thresholdPct = %d, want %d", gotThresholdPct, tt.wantThresholdPct)
			}
		})
	}
}
