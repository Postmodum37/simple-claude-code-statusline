# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Claude Code plugin that provides a custom two-line statusline. It's implemented as cross-compiled Go binaries dispatched via a thin bash shim (`bin/statusline.sh`).

## Architecture

This repo serves as both a **marketplace** and a **plugin**:

- `.claude-plugin/marketplace.json` - Makes this repo a Claude Code marketplace
- `.claude-plugin/plugin.json` - Plugin manifest
- `src/` - Go source files (stdin parsing, rendering, git, usage API, formatting)
- `bin/statusline.sh` - Bash shim that detects OS/arch and execs the correct binary
- `bin/{os}-{arch}/statusline` - Cross-compiled Go binaries (darwin/linux, amd64/arm64)
- `Makefile` - Build targets: `build` (cross-compile all), `test`, `clean`
- `commands/setup.md` - Command that configures `~/.claude/settings.json`
- `hooks/` - SessionStart hook that reminds users to run setup if not configured

When installed, the plugin runs from `${CLAUDE_PLUGIN_ROOT}/bin/statusline.sh` (the plugin cache directory), which dispatches to the platform-appropriate binary.

## How the Statusline Works

Claude Code pipes JSON to the binary via stdin containing:
- `model.id` / `model.display_name` - Current model
- `cwd` / `workspace.project_dir` - Current/project directories
- `context_window.*` - Token usage and context size
- `session_id` - For session duration tracking
- `cost.total_lines_added` / `cost.total_lines_removed` - Session-cumulative lines changed
- `agent.name` - Agent name (when using `--agent` flag)

The binary outputs two lines of ANSI-escaped text:
1. Model [agent] | Directory | Git branch + status | Session lines changed
2. Context bar | 5h rate limit | 7d rate limit | Cost | Duration

## Building

```sh
make build    # Cross-compile all platform binaries
make test     # Run Go test suite
make clean    # Remove compiled binaries
```

**When to rebuild binaries**: You MUST run `make build` and commit the resulting binaries whenever you modify any file in `src/`. The binaries under `bin/{os}-{arch}/` are checked into git and are what users actually run — source changes have no effect until binaries are rebuilt. Always include the rebuilt binaries in the same commit as the source changes.

## Testing

Run the Go test suite:
```sh
make test
```

Test manually by piping sample JSON:
```sh
echo '{"model":{"id":"claude-opus-4-6"},"cwd":"/tmp","context_window":{"used_percentage":42,"context_window_size":200000}}' | ./bin/statusline.sh
```

Note: Include `workspace.project_dir` in JSON for git info to display.

## Screenshots

Do NOT use termshot/vhs for screenshots - they render fonts incorrectly. Ask user to take manual screenshots from their terminal after running test commands with different mock JSON states.

## External Dependencies

The Go binary handles JSON parsing and HTTP natively. External commands used:
- `git` - Repository status (with `--no-optional-locks` to avoid conflicts)
- macOS `security` command - OAuth token retrieval from keychain (falls back to `~/.claude/.credentials.json`)
- `~/.claude.json` - Auto-compact setting detection

## Key Implementation Notes

- Adding a new JSON field requires updating the `StdinData` struct in `src/stdin.go` and this file's "Available JSON fields" section
- Go source is in `src/` with one package (`main`): stdin parsing, model ID parsing, formatting, git status, usage API, auto-compact detection, and ANSI rendering
- Two-phase exit: renders output to stdout first, then closes stdout and waits for background usage API fetch to complete
- Usage API fetch runs in a goroutine with 200ms wait timeout — never blocks render
- Caches to `${CLAUDE_CODE_TMPDIR:-/tmp}/claude-*` (git: 5s TTL; usage: 10min TTL with 5min 429 backoff). Atomic writes via tmpfile + rename.
- Colors use Tokyo Night palette as constants in `src/render.go`
- Lines changed shows session-cumulative totals from `cost.total_lines_added`/`cost.total_lines_removed`
- Auto-compact indicator `(↻)` shown when auto-compact is enabled
- `>200k` indicator shown when token count exceeds 200k (fast mode pricing threshold)
- Context display uses `used_percentage` as single source of truth for bar/color/percentage. `current_usage.*` drives absolute token count display only.

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

### Last reviewed: v2.1.63 (Mar 3, 2026)

**v2.1.29–v2.1.31** — No statusline-impacting changes. v2.1.31 reduced terminal layout jitter during spinner transitions, which may improve statusline rendering stability.

**v2.1.32** — Claude Opus 4.6 released (`claude-opus-4-6`). Model ID parsing handles this correctly (outputs "Opus 4.6"). Also introduced agent teams (experimental) and auto memory.

**v2.1.33** — Added `TeammateIdle`/`TaskCompleted` hook events for agent teams. Plugin name now shown in skill descriptions.

**v2.1.34–v2.1.39** — Mostly bug fixes and stability improvements. v2.1.36 added fast mode for Opus 4.6. v2.1.39 improved terminal rendering performance. No new statusline JSON fields were added; the `speed` attribute (fast mode) was added to OTel tracing only, not exposed in statusline input.

**v2.1.40** — Version number skipped in changelog.

**v2.1.41** — Narrow terminal layout improvements. `speed` attribute added to OTel (fast mode visibility) but not exposed in statusline JSON. New CLI auth subcommands.

**v2.1.42** — Startup performance improvements (deferred Zod schema). Date moved out of system prompt. Opus 4.6 effort callout.

**v2.1.43–v2.1.44** — Bug fixes only. No statusline-impacting changes.

**v2.1.45** — Claude Sonnet 4.6 released (`claude-sonnet-4-6`). Model ID parsing handles this correctly (outputs "Sonnet 4.6"). SDK gained `SDKRateLimitInfo`/`SDKRateLimitEvent` types for rate limit status — SDK-only, not yet exposed in statusline JSON. Plugins no longer require restart after installation.

**v2.1.46** — claude.ai MCP connectors support. No statusline changes.

**v2.1.47** — Added `workspace.added_dirs` to statusline JSON (directories from `/add-dir`). Not yet used by us.

**v2.1.49** — `--worktree` flag; Sonnet 4.5 1M removed (Sonnet 4.6 1M replaces it); `ConfigChange` hook; SDK model info fields (`supportsEffort`, etc.). No new statusline fields.

**v2.1.50** — Opus 4.6 fast mode gets 1M context window. Model IDs now include `[1m]` suffix for 1M context (e.g., `claude-opus-4-6[1m]`). `CLAUDE_CODE_DISABLE_1M_CONTEXT` env var. Also: `isolation: worktree` for agents, `CLAUDE_CODE_SIMPLE`, `claude agents` CLI.

**v2.1.51** — `/model` shows human-readable labels. Security fix: statusline hooks now require workspace trust. `CLAUDE_CODE_ACCOUNT_UUID`/`USER_EMAIL`/`ORGANIZATION_UUID` env vars; managed settings via plist/Registry.

**v2.1.53–v2.1.58** — Bug fixes, Windows stability, memory leak fixes. No statusline changes.

**v2.1.59** — Auto-memory with `/memory`; `/copy` command; MCP OAuth refresh race fix. No statusline changes.

**v2.1.62** — Prompt suggestion cache fix. No statusline changes.

**v2.1.63** — HTTP hooks; `/simplify` and `/batch` commands; memory leak fixes; `ENABLE_CLAUDEAI_MCP_SERVERS=false`. No statusline changes.

### Statusline JSON field changes in v2.1.29–v2.1.63

v2.1.47 added `workspace.added_dirs`. v2.1.50 introduced the `[1m]` suffix on model IDs for 1M context models (handled in `src/model.go` — we strip `[...]` before version parsing). All other statusline fields remained stable. The SDK gained rate limit types in v2.1.45 (`SDKRateLimitInfo`) but rate limit data is still not exposed in statusline JSON.

### Usage API changes (not in Claude Code changelog)

The `/api/oauth/usage` response now includes per-model rate limit fields (`seven_day_opus`, `seven_day_sonnet`, `seven_day_oauth_apps`, `seven_day_cowork`) and an unknown `iguana_necktie` field. These are additive — existing `five_hour` and `seven_day` fields are unchanged. `extra_usage.utilization` can now be `null` (Go's zero-value handles this).

### Available JSON fields not yet used

These exist in the statusline JSON but we don't leverage them:

- `version` — Claude Code version string (e.g., "2.1.63")
- `vim.mode` — current vim mode
- `output_style.name` — current output style
- `cost.total_api_duration_ms` — API time vs wall time
- `context_window.remaining_percentage` — pre-calculated remaining % (inverse of `used_percentage`)
- `transcript_path` — path to conversation transcript file
- `context_window.total_input_tokens` — cumulative input tokens across session
- `context_window.total_output_tokens` — cumulative output tokens across session
- `workspace.added_dirs` — directories added via `/add-dir` (since v2.1.47)

### Open issues to track

- [#22221](https://github.com/anthropics/claude-code/issues/22221) — Expose rate limits in statusline JSON (would eliminate our OAuth API call; open, labeled `enhancement`/`med-priority`, no assignees. SDK has rate limit types since v2.1.45 but still not in statusline input)
- [#17959](https://github.com/anthropics/claude-code/issues/17959) — `used_percentage` mismatch with Claude Code's internal "Context low" warning (marked stale by bot)
