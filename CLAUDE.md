# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Claude Code plugin that provides a custom two-line statusline. It's a pure bash script with no build system - the script runs directly when Claude Code renders its status bar.

## Architecture

This repo serves as both a **marketplace** and a **plugin**:

- `.claude-plugin/marketplace.json` - Makes this repo a Claude Code marketplace
- `.claude-plugin/plugin.json` - Plugin manifest
- `bin/statusline.sh` - The main script. Reads JSON from stdin, outputs ANSI-formatted text to stdout.
- `commands/setup.md` - Command that configures `~/.claude/settings.json`
- `hooks/` - SessionStart hook that reminds users to run setup if not configured

When installed, the plugin runs from `${CLAUDE_PLUGIN_ROOT}/bin/statusline.sh` (the plugin cache directory).

## How the Statusline Works

Claude Code pipes JSON to the script containing:
- `model.id` / `model.display_name` - Current model
- `cwd` / `workspace.project_dir` - Current/project directories
- `context_window.*` - Token usage and context size
- `session_id` - For session duration tracking
- `cost.total_lines_added` / `cost.total_lines_removed` - Session-cumulative lines changed
- `agent.name` - Agent name (when using `--agent` flag)

The script outputs two lines of ANSI-escaped text:
1. Model [agent] | Directory | Git branch + status | Session lines changed
2. Context bar | 5h rate limit | 7d rate limit | Cost | Duration

## Testing

Test manually by piping sample JSON:
```sh
echo '{"model":{"id":"claude-opus-4-5-20251101"},"cwd":"/tmp","context_window":{"used_percentage":42,"context_window_size":200000}}' | ./bin/statusline.sh
```

Note: Include `workspace.project_dir` in JSON for git info to display.

## Screenshots

Do NOT use termshot/vhs for screenshots - they render fonts incorrectly. Ask user to take manual screenshots from their terminal after running test commands with different mock JSON states.

## External Dependencies

The script uses:
- `jq` - JSON parsing (required)
- `curl` - Rate limit API calls
- `git` - Repository status
- macOS `security` command - OAuth token retrieval from keychain
- Platform-specific: `stat -f %m` (macOS) or `stat -c %Y` (Linux) for cache age
- Platform-specific: `date -j -f` (macOS) or `date -d` (Linux) for ISO date parsing
- `~/.claude.json` - Auto-compact setting detection

## Key Implementation Notes

- Adding a new JSON field requires updates in 3 places: defaults (before `eval`), jq `@sh` block (last line has no trailing comma), and CLAUDE.md "Available JSON fields" section
- Uses `--no-optional-locks` with git commands to avoid conflicts
- Caches to `${CLAUDE_CODE_TMPDIR:-/tmp}/claude-*` (rate limit: 60s TTL, 15s for errors; git: 5s TTL)
- Colors use Tokyo Night palette defined at top of script
- Compatible with bash 3.2 (macOS default) - uses `=~` without capture groups to avoid `BASH_REMATCH`
- Cross-platform: auto-detects macOS vs Linux for stat/date commands
- Lines changed shows session-cumulative totals from `cost.total_lines_added`/`cost.total_lines_removed`
- Cache token count displayed when prompt caching is active
- Auto-compact indicator `(↻)` shown when auto-compact is enabled

## Plugin Development

Use `plugin-dev` (Anthropic's official plugin development toolkit) to validate changes:
- `plugin-dev:plugin-validator` - Validates plugin structure, manifests, and commands
- `plugin-dev:skill-reviewer` - Reviews skills if added

## Commits

Use `/commit-commands:commit` for commits. Follow conventional commit style:
- `feat:` new features
- `fix:` bug fixes
- `docs:` documentation changes
- `chore:` maintenance tasks

## Versioning

Bump version in both `marketplace.json` and `.claude-plugin/plugin.json` for:
- `feat:` - new features
- `fix:` - bug fixes
- `chore:` - maintenance tasks

Do NOT bump version for:
- `docs:` - documentation-only changes (README, CLAUDE.md, comments)

## Claude Code Version Reviews

Track which Claude Code versions have been reviewed for statusline-relevant changes.

### Last reviewed: v2.1.33 (Feb 6, 2026)

**v2.1.29–v2.1.31** — No statusline-impacting changes. v2.1.31 reduced terminal layout jitter during spinner transitions, which may improve statusline rendering stability.

**v2.1.32** — Claude Opus 4.6 released (`claude-opus-4-6`). Model ID parsing handles this correctly (outputs "Opus 4.6"). Also introduced agent teams (experimental) and auto memory.

**v2.1.33** — Added `TeammateIdle`/`TaskCompleted` hook events for agent teams. Plugin name now shown in skill descriptions.

**v2.1.34** — Current version, no public changelog yet as of Feb 6, 2026.

### No new statusline JSON fields in v2.1.29–v2.1.33

The statusline input schema remained stable across these versions.

### Available JSON fields not yet used

These exist in the statusline JSON but we don't leverage them:

- `version` — Claude Code version string (e.g., "2.1.33")
- `vim.mode` — current vim mode
- `output_style.name` — current output style
- `cost.total_api_duration_ms` — API time vs wall time

### Open issues to track

- [#22221](https://github.com/anthropics/claude-code/issues/22221) — Expose rate limits in statusline JSON (would eliminate our OAuth API call)
- [#17959](https://github.com/anthropics/claude-code/issues/17959) — `used_percentage` mismatch with Claude Code's internal "Context low" warning
