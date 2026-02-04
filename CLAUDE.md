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

The script outputs two lines of ANSI-escaped text:
1. Model | Directory | Git branch + status | Lines changed
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

- Uses `--no-optional-locks` with git commands to avoid conflicts
- Caches to `${CLAUDE_CODE_TMPDIR:-/tmp}/claude-*` (rate limit: 60s TTL, 15s for errors; git: 5s TTL)
- Colors use Tokyo Night palette defined at top of script
- Compatible with bash 3.2 (macOS default) - uses `=~` without capture groups to avoid `BASH_REMATCH`
- Cross-platform: auto-detects macOS vs Linux for stat/date commands
- Lines changed shows current uncommitted changes (`git diff HEAD`), not cumulative session edits
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
