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
		modelID           string
		claudeJSON        string   // legacy ~/.claude.json contents; empty = file absent
		settings          []string // settings.json contents, lowest → highest precedence; "" = file absent
		envVars           map[string]string
		wantEnabled       bool
		wantThresholdPct  int
	}{
		{
			name:              "200k context, auto-compact enabled by default (no files)",
			contextWindowSize: 200000,
			wantEnabled:       true,
			wantThresholdPct:  83, // (200000-20000-13000)*100/200000 = 83.5 → 83
		},
		{
			name:              "200k context, legacy field absent",
			contextWindowSize: 200000,
			claudeJSON:        `{"theme": "dark"}`,
			wantEnabled:       true,
			wantThresholdPct:  83,
		},
		{
			name:              "200k context, legacy explicitly enabled",
			contextWindowSize: 200000,
			claudeJSON:        `{"autoCompactEnabled": true}`,
			wantEnabled:       true,
			wantThresholdPct:  83,
		},
		{
			name:              "1M context, no overrides",
			contextWindowSize: 1000000,
			wantEnabled:       true,
			wantThresholdPct:  96, // (1000000-20000-13000)*100/1000000 = 96.7 → 96
		},
		{
			name:              "200k context, CLAUDE_AUTOCOMPACT_PCT_OVERRIDE=70",
			contextWindowSize: 200000,
			envVars:           map[string]string{"CLAUDE_AUTOCOMPACT_PCT_OVERRIDE": "70"},
			wantEnabled:       true,
			wantThresholdPct:  63, // 70% of 180000 = 126000 → 63%
		},
		{
			name:              "200k context, CLAUDE_AUTOCOMPACT_PCT_OVERRIDE=99 (cannot raise the threshold)",
			contextWindowSize: 200000,
			envVars:           map[string]string{"CLAUDE_AUTOCOMPACT_PCT_OVERRIDE": "99"},
			wantEnabled:       true,
			wantThresholdPct:  83,
		},
		{
			name:              "200k context, CLAUDE_CODE_AUTO_COMPACT_WINDOW=100000",
			contextWindowSize: 200000,
			envVars:           map[string]string{"CLAUDE_CODE_AUTO_COMPACT_WINDOW": "100000"},
			wantEnabled:       true,
			wantThresholdPct:  33, // window=100000, effective=80000, threshold=67000 → 33%
		},
		{
			name:              "CLAUDE_CODE_AUTO_COMPACT_WINDOW below 100k is clamped up to 100k",
			contextWindowSize: 200000,
			envVars:           map[string]string{"CLAUDE_CODE_AUTO_COMPACT_WINDOW": "10000"},
			wantEnabled:       true,
			wantThresholdPct:  33,
		},
		{
			name:              "CLAUDE_CODE_AUTO_COMPACT_WINDOW larger than model window is a no-op",
			contextWindowSize: 200000,
			envVars:           map[string]string{"CLAUDE_CODE_AUTO_COMPACT_WINDOW": "500000"},
			wantEnabled:       true,
			wantThresholdPct:  83,
		},
		{
			name:              "legacy ~/.claude.json disables auto-compact",
			contextWindowSize: 200000,
			claudeJSON:        `{"autoCompactEnabled": false}`,
			wantEnabled:       false,
		},
		{
			name:              "user settings.json disables auto-compact",
			contextWindowSize: 200000,
			settings:          []string{`{"autoCompactEnabled": false}`},
			wantEnabled:       false,
		},
		{
			name:              "settings.json true overrides legacy false",
			contextWindowSize: 200000,
			claudeJSON:        `{"autoCompactEnabled": false}`,
			settings:          []string{`{"autoCompactEnabled": true}`},
			wantEnabled:       true,
			wantThresholdPct:  83,
		},
		{
			name:              "higher-precedence settings file wins",
			contextWindowSize: 200000,
			settings:          []string{`{"autoCompactEnabled": true}`, `{"autoCompactEnabled": false}`},
			wantEnabled:       false,
		},
		{
			name:              "lower-precedence value used when higher file lacks the key",
			contextWindowSize: 200000,
			settings:          []string{`{"autoCompactEnabled": false}`, `{"theme": "dark"}`},
			wantEnabled:       false,
		},
		{
			name:              "DISABLE_AUTO_COMPACT env kill switch",
			contextWindowSize: 200000,
			envVars:           map[string]string{"DISABLE_AUTO_COMPACT": "1"},
			wantEnabled:       false,
		},
		{
			name:              "DISABLE_COMPACT env kill switch",
			contextWindowSize: 200000,
			envVars:           map[string]string{"DISABLE_COMPACT": "true"},
			wantEnabled:       false,
		},
		{
			name:              "autoCompactWindow setting caps a 1M window",
			contextWindowSize: 1000000,
			settings:          []string{`{"autoCompactWindow": 500000}`},
			wantEnabled:       true,
			wantThresholdPct:  46, // (500000-20000-13000)*100/1000000 = 46.7 → 46
		},
		{
			name:              "autoCompactWindow setting out of range is ignored",
			contextWindowSize: 1000000,
			settings:          []string{`{"autoCompactWindow": 50000}`},
			wantEnabled:       true,
			wantThresholdPct:  96,
		},
		{
			name:              "env window takes precedence over autoCompactWindow setting",
			contextWindowSize: 1000000,
			settings:          []string{`{"autoCompactWindow": 500000}`},
			envVars:           map[string]string{"CLAUDE_CODE_AUTO_COMPACT_WINDOW": "200000"},
			wantEnabled:       true,
			wantThresholdPct:  16, // (200000-20000-13000)*100/1000000 = 16.7 → 16
		},
		{
			name:              "server-provided per-model window (number) from legacy cache",
			contextWindowSize: 1000000,
			modelID:           "claude-fable-5-1[1m]",
			claudeJSON:        `{"autoCompactWindowsCache": {"claude-fable-5-1[1m]": 500000}}`,
			wantEnabled:       true,
			wantThresholdPct:  46,
		},
		{
			name:              "server-provided per-model window (object default) from legacy cache",
			contextWindowSize: 1000000,
			modelID:           "claude-sonnet-5",
			claudeJSON:        `{"autoCompactWindowsCache": {"claude-sonnet-5": {"default": 500000, "surfaces": {"remote_cowork": {"default": 300000}}}}}`,
			wantEnabled:       true,
			wantThresholdPct:  46,
		},
		{
			name:              "per-model cache for a different model is ignored",
			contextWindowSize: 1000000,
			modelID:           "claude-opus-5",
			claudeJSON:        `{"autoCompactWindowsCache": {"claude-sonnet-5": 500000}}`,
			wantEnabled:       true,
			wantThresholdPct:  96,
		},
		{
			name:              "autoCompactWindow setting beats per-model cache",
			contextWindowSize: 1000000,
			modelID:           "claude-sonnet-5",
			claudeJSON:        `{"autoCompactWindowsCache": {"claude-sonnet-5": 300000}}`,
			settings:          []string{`{"autoCompactWindow": 500000}`},
			wantEnabled:       true,
			wantThresholdPct:  46,
		},
		{
			name:              "CLAUDE_CODE_MAX_OUTPUT_TOKENS below 20k shrinks the output reserve",
			contextWindowSize: 200000,
			envVars:           map[string]string{"CLAUDE_CODE_MAX_OUTPUT_TOKENS": "8000"},
			wantEnabled:       true,
			wantThresholdPct:  89, // (200000-8000-13000)*100/200000 = 89.5 → 89
		},
		{
			name:              "CLAUDE_CODE_MAX_OUTPUT_TOKENS above 20k does not grow the reserve",
			contextWindowSize: 200000,
			envVars:           map[string]string{"CLAUDE_CODE_MAX_OUTPUT_TOKENS": "64000"},
			wantEnabled:       true,
			wantThresholdPct:  83,
		},
		{
			name:              "degenerate: context window too small",
			contextWindowSize: 20000,
			wantEnabled:       false,
		},
		{
			name:              "degenerate: zero context window",
			contextWindowSize: 0,
			wantEnabled:       false,
		},
		{
			name:              "invalid CLAUDE_AUTOCOMPACT_PCT_OVERRIDE ignored",
			contextWindowSize: 200000,
			envVars:           map[string]string{"CLAUDE_AUTOCOMPACT_PCT_OVERRIDE": "abc"},
			wantEnabled:       true,
			wantThresholdPct:  83,
		},
		{
			name:              "CLAUDE_AUTOCOMPACT_PCT_OVERRIDE=0 ignored",
			contextWindowSize: 200000,
			envVars:           map[string]string{"CLAUDE_AUTOCOMPACT_PCT_OVERRIDE": "0"},
			wantEnabled:       true,
			wantThresholdPct:  83,
		},
		{
			name:              "CLAUDE_AUTOCOMPACT_PCT_OVERRIDE=101 ignored",
			contextWindowSize: 200000,
			envVars:           map[string]string{"CLAUDE_AUTOCOMPACT_PCT_OVERRIDE": "101"},
			wantEnabled:       true,
			wantThresholdPct:  83,
		},
		{
			name:              "invalid legacy JSON defaults to enabled",
			contextWindowSize: 200000,
			claudeJSON:        `not valid json`,
			wantEnabled:       true,
			wantThresholdPct:  83,
		},
		{
			name:              "invalid settings JSON is skipped",
			contextWindowSize: 200000,
			settings:          []string{`{"autoCompactEnabled": false}`, `{not json`},
			wantEnabled:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Neutralize any of the relevant env vars inherited from the host.
			for _, k := range []string{"CLAUDE_AUTOCOMPACT_PCT_OVERRIDE", "CLAUDE_CODE_AUTO_COMPACT_WINDOW", "CLAUDE_CODE_MAX_OUTPUT_TOKENS", "DISABLE_AUTO_COMPACT", "DISABLE_COMPACT"} {
				t.Setenv(k, "")
			}
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			dir := t.TempDir()
			cfg := CompactConfig{
				ClaudeJSONPath: filepath.Join(dir, "claude.json"),
				ModelID:        tt.modelID,
			}
			if tt.claudeJSON != "" {
				if err := os.WriteFile(cfg.ClaudeJSONPath, []byte(tt.claudeJSON), 0644); err != nil {
					t.Fatal(err)
				}
			}
			for i, content := range tt.settings {
				path := filepath.Join(dir, "settings-"+string(rune('a'+i))+".json")
				cfg.SettingsPaths = append(cfg.SettingsPaths, path)
				if content != "" {
					if err := os.WriteFile(path, []byte(content), 0644); err != nil {
						t.Fatal(err)
					}
				}
			}
			// Always include a nonexistent path to prove missing files are harmless.
			cfg.SettingsPaths = append(cfg.SettingsPaths, filepath.Join(dir, "missing", "settings.json"))

			gotEnabled, gotThresholdPct := GetCompactThreshold(tt.contextWindowSize, cfg)
			if gotEnabled != tt.wantEnabled {
				t.Errorf("enabled = %v, want %v", gotEnabled, tt.wantEnabled)
			}
			if gotThresholdPct != tt.wantThresholdPct {
				t.Errorf("thresholdPct = %d, want %d", gotThresholdPct, tt.wantThresholdPct)
			}
		})
	}
}

func TestDefaultCompactConfig(t *testing.T) {
	cfg := DefaultCompactConfig("/home/u", "/repo", "claude-opus-5")
	if cfg.ClaudeJSONPath != "/home/u/.claude.json" {
		t.Errorf("ClaudeJSONPath = %q", cfg.ClaudeJSONPath)
	}
	if cfg.ModelID != "claude-opus-5" {
		t.Errorf("ModelID = %q", cfg.ModelID)
	}
	want := []string{"/home/u/.claude/settings.json", "/repo/.claude/settings.json", "/repo/.claude/settings.local.json"}
	if len(cfg.SettingsPaths) < len(want) {
		t.Fatalf("SettingsPaths = %v, want at least %v", cfg.SettingsPaths, want)
	}
	for i, w := range want {
		if cfg.SettingsPaths[i] != w {
			t.Errorf("SettingsPaths[%d] = %q, want %q", i, cfg.SettingsPaths[i], w)
		}
	}

	noProject := DefaultCompactConfig("/home/u", "", "")
	if len(noProject.SettingsPaths) != len(cfg.SettingsPaths)-2 {
		t.Errorf("expected fewer paths without a project dir: %v", noProject.SettingsPaths)
	}
}
