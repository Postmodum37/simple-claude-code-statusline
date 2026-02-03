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
1. Model | Directory | Git branch + status | PR status | Lines changed
2. Context bar | 5h rate limit | 7d rate limit | Cost | Duration

## Testing

Test manually by piping sample JSON:
```sh
echo '{"model":{"id":"claude-opus-4-5-20251101"},"cwd":"/tmp","context_window":{"used_percentage":42,"context_window_size":200000}}' | ./bin/statusline.sh
```

## External Dependencies

The script uses:
- `jq` - JSON parsing (required)
- `curl` - Rate limit API calls
- `git` - Repository status
- `gh` - GitHub CLI for PR status (optional - PR status hidden if not installed)
- macOS `security` command - OAuth token retrieval from keychain
- Platform-specific: `stat -f %m` (macOS) or `stat -c %Y` (Linux) for cache age
- Platform-specific: `date -j -f` (macOS) or `date -d` (Linux) for ISO date parsing
- `~/.claude.json` - Auto-compact setting detection

## Configuration

Environment variables can customize behavior:

- `STATUSLINE_SHOW_PR` - Set to `false` to hide PR status (default: `true`). Useful if you prefer Claude Code's native PR indicator (added in 2.1.20+).

## Key Implementation Notes

- Uses `--no-optional-locks` with git commands to avoid conflicts
- Caches to `${CLAUDE_CODE_TMPDIR:-/tmp}/claude-*` (rate limit: 60s TTL, 15s for errors; git: 5s TTL; PR: 30s TTL)
- Colors use Tokyo Night palette defined at top of script
- Compatible with bash 3.2 (macOS default) - uses `=~` without capture groups to avoid `BASH_REMATCH`
- Cross-platform: auto-detects macOS vs Linux for stat/date commands
- PR merged status uses purple color to match Claude Code 2.1.23+ styling
- Lines changed shows current uncommitted changes (`git diff HEAD`), not cumulative session edits
- TPM (Tokens Per Minute) displayed after 30+ seconds of session activity
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
