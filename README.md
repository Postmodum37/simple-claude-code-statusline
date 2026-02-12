# Simple Claude Code Statusline

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A minimal, hackable two-line statusline for Claude Code.

![Two-line statusline showing model, git status, and context usage](screenshot.png)

## Features

**Line 1:** Model [agent] | Directory | Git branch + status | Session lines changed
**Line 2:** Context bar | 5h rate limit | 7d rate limit | Cost | Duration

- Tokyo Night color scheme
- Context usage with color-coded progress bar
- Rate limit tracking with time until reset
- Git branch with added/modified/deleted counts and ahead/behind tracking
- Git worktree support with `[wt:name]` indicator
- Agent name display when using `--agent` flag
- Session lines changed (cumulative +added/-removed)
- Session cost tracking ($X.XX)
- Auto-compact indicator (↻) when enabled
- `>200k` token threshold indicator (fast mode pricing doubles past 200k)
- Cross-platform (macOS and Linux)
- No build step - just bash

### Context Usage Colors

| Usage | Color | Meaning |
|-------|-------|---------|
| 0-50% | Green | Plenty of context remaining |
| 51-75% | Yellow | Getting full |
| 76-90% | Orange | Consider summarizing |
| 91%+ | Red | Near limit |

### Git Features

![Git branch with status indicators](screenshot-git.png)

- **Branch name** with file status counts (✚added/●modified/✖deleted)
- **Ahead/behind** tracking: `↑2` commits ahead, `↓1` behind upstream
- **Worktree indicator**: `[wt:feature-name]` when in a linked worktree
- **Session lines changed**: `+44/-14` cumulative lines added/removed this session

### Model Display

![Sonnet model with yellow context bar](screenshot-sonnet.png)

Shows abbreviated model names: Opus 4.6, Sonnet 4.5, Haiku, etc.

## Requirements

- `jq` - JSON parsing
- `curl` - Rate limit API calls
- `git` - Repository status (optional)

On macOS, the script also uses the `security` command to retrieve OAuth tokens from keychain.

Install dependencies on macOS:
```sh
brew install jq
```

## Installation

### Option 1: Plugin (recommended)

Add the marketplace:
```sh
/plugin marketplace add Postmodum37/simple-claude-code-statusline
```

Install the plugin:
```sh
/plugin install simple-statusline
```

Restart Claude Code, then configure:
```sh
/simple-statusline:setup
```

The statusline appears immediately after setup (no second restart needed).

### Option 2: Manual

Copy the script:
```sh
curl -o ~/.claude/statusline.sh https://raw.githubusercontent.com/Postmodum37/simple-claude-code-statusline/main/bin/statusline.sh
chmod +x ~/.claude/statusline.sh
```

Add to `~/.claude/settings.json`:
```json
{
  "statusLine": {
    "type": "command",
    "command": "~/.claude/statusline.sh"
  }
}
```

Restart Claude Code.

## Customization

Edit `~/.claude/statusline.sh` directly. The script is self-contained and well-commented.

### Colors

Tokyo Night palette defined at the top:
```bash
C_ACCENT="\033[38;5;111m"     # Blue - model/branch
C_MUTED="\033[38;5;146m"      # Gray - separators
C_OK="\033[38;5;114m"         # Green - 0-50%
C_WARN="\033[38;5;214m"       # Yellow - 51-75%
C_HIGH="\033[38;5;208m"       # Orange - 76-90%
C_CRIT="\033[38;5;196m"       # Red - 91%+
```

### Layout

Modify the two-row output at the bottom of the script:
```bash
row1="$seg_model"
[[ -n "$dir_display" ]] && row1+="${sep}${seg_dir}"
[[ -n "$git_branch" ]] && row1+="${sep}${seg_git}"

row2="$seg_context"
[[ -n "$usage_5h" ]] && row2+="${sep}${seg_usage_5h}"
[[ -n "$usage_7d" ]] && row2+="${sep}${seg_usage_7d}"
row2+="${sep}${seg_duration}"
```

## JSON Input Reference

Claude Code pipes JSON to statusline scripts via stdin. Here's what's available (as of Claude Code v2.1.39):

```json
{
  "hook_event_name": "Status",
  "session_id": "abc123...",
  "cwd": "/current/working/directory",
  "version": "2.1.39",
  "model": {
    "id": "claude-opus-4-6",
    "display_name": "Opus 4.6"
  },
  "workspace": {
    "current_dir": "/current/working/directory",
    "project_dir": "/original/project/directory"
  },
  "cost": {
    "total_cost_usd": 0.05,
    "total_duration_ms": 120000,
    "total_lines_added": 156,
    "total_lines_removed": 23,
    "total_api_duration_ms": 95000
  },
  "context_window": {
    "context_window_size": 200000,
    "used_percentage": 45,
    "remaining_percentage": 55,
    "current_usage": {
      "input_tokens": 50000,
      "output_tokens": 20000,
      "cache_creation_input_tokens": 10000,
      "cache_read_input_tokens": 10000
    }
  },
  "exceeds_200k_tokens": false,
  "transcript_path": "/path/to/transcript.jsonl",
  "agent": {
    "name": "my-agent"
  }
}
```

| Field | Description |
|-------|-------------|
| `model.id` / `model.display_name` | Current model identifier and display name |
| `cwd` / `workspace.project_dir` | Current working directory and project root |
| `context_window.used_percentage` | Percentage of context used (0-100) |
| `context_window.remaining_percentage` | Percentage remaining (0-100) |
| `context_window.current_usage.*` | Per-API-call token breakdown by type |
| `exceeds_200k_tokens` | Whether token count exceeds 200k (fast mode pricing threshold) |
| `cost.total_lines_added` / `cost.total_lines_removed` | Session-cumulative lines changed |
| `cost.total_cost_usd` / `cost.total_duration_ms` | Session cost and duration |
| `agent.name` | Agent name when using `--agent` flag |

### Rate Limits (via OAuth API)

This plugin also fetches rate limit data from the Anthropic API (requires OAuth authentication):

- **macOS:** Uses keychain (`security find-generic-password`)
- **Linux:** Reads from `~/.claude/.credentials.json`

The API returns 5-hour and 7-day utilization percentages with reset times.

## Testing

Test the script manually:
```sh
echo '{"model":{"id":"claude-opus-4-5-20251101"},"cwd":"/tmp","context_window":{"used_percentage":42,"context_window_size":200000},"cost":{"total_duration_ms":3600000,"total_lines_added":50,"total_lines_removed":10}}' | ~/.claude/statusline.sh
```

## Uninstalling

Clean up the statusline config first:
```sh
/simple-statusline:cleanup
```

Then uninstall the plugin:
```sh
/plugin uninstall simple-statusline
```

## License

MIT
