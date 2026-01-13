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
1. Model | Directory | Git branch + status
2. Context bar | 5h rate limit | 7d rate limit | Duration

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
- macOS `security` command - OAuth token retrieval from keychain
- macOS `stat -f` syntax - Cache age checking

## Key Implementation Notes

- Uses `--no-optional-locks` with git commands to avoid conflicts
- Caches rate limit API responses to `/tmp/claude-usage-cache` (60s TTL)
- Session start time stored in `/tmp/claude-session-{id}`
- Colors use Tokyo Night palette defined at top of script
- Compatible with bash 3.2 (macOS default) - avoids `BASH_REMATCH`

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
