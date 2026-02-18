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
- Auto-compact indicator `(↻)` shown when auto-compact is enabled
- `>200k` indicator shown when token count exceeds 200k (fast mode pricing threshold)

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

### Last reviewed: v2.1.45 (Feb 18, 2026)

**v2.1.29–v2.1.31** — No statusline-impacting changes. v2.1.31 reduced terminal layout jitter during spinner transitions, which may improve statusline rendering stability.

**v2.1.32** — Claude Opus 4.6 released (`claude-opus-4-6`). Model ID parsing handles this correctly (outputs "Opus 4.6"). Also introduced agent teams (experimental) and auto memory.

**v2.1.33** — Added `TeammateIdle`/`TaskCompleted` hook events for agent teams. Plugin name now shown in skill descriptions.

**v2.1.34–v2.1.39** — Mostly bug fixes and stability improvements. v2.1.36 added fast mode for Opus 4.6. v2.1.39 improved terminal rendering performance. No new statusline JSON fields were added; the `speed` attribute (fast mode) was added to OTel tracing only, not exposed in statusline input.

**v2.1.40** — Version number skipped in changelog.

**v2.1.41** — Narrow terminal layout improvements. `speed` attribute added to OTel (fast mode visibility) but not exposed in statusline JSON. New CLI auth subcommands.

**v2.1.42** — Startup performance improvements (deferred Zod schema). Date moved out of system prompt. Opus 4.6 effort callout.

**v2.1.43–v2.1.44** — Bug fixes only. No statusline-impacting changes.

**v2.1.45** — Claude Sonnet 4.6 released (`claude-sonnet-4-6`). Model ID parsing handles this correctly (outputs "Sonnet 4.6"). SDK gained `SDKRateLimitInfo`/`SDKRateLimitEvent` types for rate limit status — SDK-only, not yet exposed in statusline JSON. Plugins no longer require restart after installation.

### No new statusline JSON fields in v2.1.29–v2.1.45

The statusline input schema remained stable across these versions. The official statusline docs now document several fields (`context_window.current_usage`, `context_window.remaining_percentage`, `exceeds_200k_tokens`, `transcript_path`) — we now use `exceeds_200k_tokens` for the >200k indicator. The SDK gained rate limit types in v2.1.45 (`SDKRateLimitInfo`), suggesting rate limit data may eventually be exposed to statusline (tracked in #22221).

### Available JSON fields not yet used

These exist in the statusline JSON but we don't leverage them:

- `version` — Claude Code version string (e.g., "2.1.39")
- `vim.mode` — current vim mode
- `output_style.name` — current output style
- `cost.total_api_duration_ms` — API time vs wall time
- `context_window.current_usage` — object with per-API-call token counts (`input_tokens`, `output_tokens`, `cache_creation_input_tokens`, `cache_read_input_tokens`)
- `context_window.remaining_percentage` — pre-calculated remaining % (inverse of `used_percentage`)
- `transcript_path` — path to conversation transcript file
- `context_window.total_input_tokens` — cumulative input tokens across session
- `context_window.total_output_tokens` — cumulative output tokens across session

### Open issues to track

- [#22221](https://github.com/anthropics/claude-code/issues/22221) — Expose rate limits in statusline JSON (would eliminate our OAuth API call; open, labeled `enhancement`/`med-priority`, no assignees. SDK now has rate limit types as of v2.1.45)
- [#17959](https://github.com/anthropics/claude-code/issues/17959) — `used_percentage` mismatch with Claude Code's internal "Context low" warning (marked stale by bot)
