# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Claude Code plugin that provides a custom two-line statusline. It's implemented as cross-compiled Go binaries dispatched via a thin bash shim (`bin/statusline.sh`).

## Architecture

This repo serves as both a **marketplace** and a **plugin**:

- `.claude-plugin/marketplace.json` - Makes this repo a Claude Code marketplace
- `.claude-plugin/plugin.json` - Plugin manifest
- `src/` - Go source files (stdin parsing, model names, rendering, git, auto-compact, formatting)
- `bin/statusline.sh` - Bash shim that detects OS/arch and execs the correct binary
- `bin/{os}-{arch}/statusline` - Cross-compiled Go binaries (darwin/linux, amd64/arm64)
- `Makefile` - Build targets: `build` (cross-compile all), `test`, `clean`
- `commands/setup.md` - Command that configures `~/.claude/settings.json`
- `hooks/` - SessionStart hook that reminds users to run setup if not configured
- `docs/claude-code-version-reviews.md` - Log of Claude Code releases reviewed for statusline impact

When installed, the plugin runs from `${CLAUDE_PLUGIN_ROOT}/bin/statusline.sh` (the plugin cache directory), which dispatches to the platform-appropriate binary.

## How the Statusline Works

Claude Code pipes JSON to the binary via stdin. Fields we display (only these are declared in `StdinData`):
- `model.id` (`model.display_name` as fallback) - Current model
- `cwd` / `workspace.project_dir` - Current/project directories
- `workspace.git_worktree` - Linked git worktree name (since v2.1.97; absent in the main working tree)
- `context_window.{used_percentage,total_input_tokens,context_window_size}` - Context usage (percentage and tokens are null/0 until the first API response)
- `cost.{total_cost_usd,total_duration_ms,total_lines_added,total_lines_removed}` - Session cost, wall time, lines changed
- `agent.name` - Agent name (when using `--agent` flag)
- `worktree.name` - Worktree name (only present in `--worktree` sessions, since v2.1.69)
- `rate_limits.{five_hour,seven_day}.{used_percentage,resets_at}` - Rate limit windows (since v2.1.80; Pro/Max only; a window is dropped once its `resets_at` passes)
- `effort.level` - Reasoning effort: `low|medium|high|xhigh|max` (since v2.1.119; the binary also still emits `auto`)
- `thinking.enabled` - `true` unless the user turned thinking off (can't be turned off on Opus 5.5/Fable)
- `pr.{number,review_state,kind}` - Open PR for the current branch (since v2.1.145; absent until a PR is found; `kind: "mr"` for GitLab merge requests)
- `fast_mode` - Whether fast mode is on

Everything else (`exceeds_200k_tokens`, `prompt_cache.*`, `rate_limits.spend_limit`, `workspace.repo`, …) is listed under "Available JSON fields not yet used" below.

The binary outputs two lines of ANSI-escaped text:
1. Model [⚡fast] [∅ thinking off] [•effort] [agent] | Directory | Git branch + status + PR badge | Session lines changed
2. Context bar + auto-compact marker | 5h rate limit | 7d rate limit | Cost | Duration

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
echo '{"model":{"id":"claude-opus-5-5"},"cwd":"/tmp","context_window":{"used_percentage":42,"total_input_tokens":84000,"context_window_size":200000}}' | ./bin/statusline.sh
```

Note: Include `workspace.project_dir` in JSON for git info to display.

## Screenshots

Three screenshots in the repo root are referenced by README.md:

- `screenshot.png` — Main hero image: Opus 5.5 on a 1M window, effort, rate limits, lines changed
- `screenshot-git.png` — Git features: longer context, git status + PR badge visible
- `screenshot-sonnet.png` — Sonnet 5, thinking off (`∅`), low context, no lines changed

**How to update:**

1. Pipe mock JSON to the binary to generate ANSI output for each scenario:
   ```sh
   echo '{"model":{"id":"claude-opus-5-5"},"cwd":"/path/to/repo","workspace":{"project_dir":"/path/to/repo"},"context_window":{"used_percentage":12,"total_input_tokens":120000,"context_window_size":1000000},"effort":{"level":"xhigh"},"thinking":{"enabled":true},"cost":{"total_cost_usd":1.25,"total_duration_ms":420000,"total_lines_added":51,"total_lines_removed":14}}' | ./bin/statusline.sh
   ```
2. Write a Python script (Pillow) that captures ANSI output to files, then renders each to PNG:
   - Parse ANSI escape codes (`\x1b[38;5;Nm`) to extract xterm-256 colors
   - Use a monospace font (SFMono or Menlo) on a Tokyo Night dark background (`#1a1b26`)
   - Render block characters (░▒▓) with tighter spacing (~92% of char width) to avoid gaps in the progress bar
   - Regular text uses standard monospace char-width advancement
3. Update README alt text if the screenshot content changes (e.g., bar color)

Do NOT use termshot/vhs — they render fonts incorrectly. The `workspace.project_dir` field must point to a real git repo for git info to appear.

## External Dependencies

External commands and files used:
- `git` - One `git status --porcelain=v2 --branch` per cache miss (branch, ahead/behind, file counts), plus `rev-parse --abbrev-ref HEAD` only if status fails; all with `--no-optional-locks` to avoid conflicts
- `~/.claude/settings.json`, `<project>/.claude/settings.json`, `<project>/.claude/settings.local.json`, managed-settings.json - `autoCompactEnabled` and `autoCompactWindow` (highest-precedence file wins)
- `~/.claude.json` - Legacy `autoCompactEnabled` fallback and the server-provided `autoCompactWindowsCache` per-model window table

## Key Implementation Notes

- `StdinData` in `src/stdin.go` declares only the fields we display; `encoding/json` ignores the rest. Adding a displayed field means adding it there and moving it out of "Available JSON fields not yet used" below.
- Go source is in `src/` with one package (`main`): stdin parsing, model ID parsing, formatting, git status, auto-compact detection, and ANSI rendering. Rate limits render straight from `stdin.RateLimits` (epoch `resets_at` → `time.Time`).
- Model names come from `model.id` (`ModelDisplayName()` in `src/model.go`), not `display_name`, which managed model-picker labels can override. Provider forms are normalized first: text before `claude-` is dropped (Bedrock `us.anthropic.`), and the ID is cut at `[`, `@`, or `:` (`[1m]`, Vertex `@20251101`, Bedrock `:0`).
- Caches to `${CLAUDE_CODE_TMPDIR:-/tmp}/claude-*` (git: 5s TTL). Atomic writes via tmpfile + rename.
- Colors use Tokyo Night palette as constants in `src/render.go`; `paint(color, s)` wraps text. Semantic colors are `level`s (ok/warn/high/crit) combined with `max`/`min` and mapped through `levelColor`. Effort levels use a separate gradient via `effortColor()`: low=muted, medium=white, high=warn, xhigh=high, max=crit, auto=accent.
- Fast mode (`fast_mode: true`) shows as a yellow `⚡` immediately after the model name; a muted `∅` follows only when `thinking.enabled` is false (thinking on is the default, so it isn't marked); effort shows as `•{level}` after that, all before the agent bracket.
- Context color (`contextColor()` in `src/render.go`) is the worse of the percent band (`pctLevel()`: ≤50/≤75/≤90) and an absolute-token band (`tokenLevel()`: ≤150k/≤300k/≤400k). On ≤200k windows this equals the percent band; on 1M windows it turns yellow at 150k, orange at 300k, red at 400k. Rationale: quality degrades with absolute context length (Pocock ~100–150k safe zone; Claude Code team's Thariq Shihipar: rot around 300–400k on 1M). The auto-compact marker is separate and purely mechanical.
- Rate-limit color (`usageColor()`) is pace-based: projected end-of-window usage = used% × window / elapsed, where elapsed is derived from `resets_at` (5h = 18000s, 7d = 604800s) and clamped to ≥25% of the window (`paceMinElapsedFraction`). Pace bands: <85 ok, <100 warn, <120 high, else crit. Pace may relax the raw percent band (`pctLevel()`) freely but raise it by at most one step (so ≤50% used is never worse than warn, ≤75% never worse than high); floored at high/crit at 75%/90% used. Falls back to percent bands without a reset time.
- Lines changed shows session-cumulative totals from `cost.total_lines_added`/`cost.total_lines_removed`
- Auto-compact indicator `(↻N%)` shown when auto-compact is enabled. `GetCompactThreshold()` in `src/compact.go` replicates Claude Code's resolution (verified against the v2.1.280 binary): kill switches `DISABLE_AUTO_COMPACT`/`DISABLE_COMPACT`; `autoCompactEnabled` from settings.json (managed > local > project > user) then legacy `~/.claude.json`; window = min(model window, `CLAUDE_CODE_AUTO_COMPACT_WINDOW` (clamped 100k–1M) → `autoCompactWindow` setting (set by `/autocompact` or `--autocompact`) → `autoCompactWindowsCache[model]` in `~/.claude.json`); effective = window − min(`CLAUDE_CODE_MAX_OUTPUT_TOKENS`, 20000); threshold = effective − 13000, lowered (never raised) by `CLAUDE_AUTOCOMPACT_PCT_OVERRIDE`. Not replicated: server-side statsig overrides and the `d1()`/experiment 200k caps for some 1M models — the JSON exposes no effective-window field.
- Git (`src/git.go`) runs a single `git status --porcelain=v2 --branch`: `# branch.head` (detached → `HEAD`), `# branch.ab`, and entry lines (`?` added, `u` conflict → modified, `1`/`2` classified by XY). Empty output means it failed (not a repo, or a large repo hitting the 1s timeout); it then falls back to `git rev-parse --abbrev-ref HEAD` so the branch still shows (and is cached), and returns nil only if that fails too.
- Worktree tag `[wt:name]` uses `worktree.name` (`--worktree` sessions), else `workspace.git_worktree`. We no longer detect worktrees ourselves.
- PR badge (`prBadge()` in `src/render.go`) renders `#<number>` (or `!<number>` when `pr.kind == "mr"`, matching Claude Code's own GitLab footer) plus a review-state glyph (✓ approved / ⏳ pending / ✗ changes_requested / ◌ draft) inside the git segment; requires git info to be present
- Context display: `used_percentage` drives the bar and color; `total_input_tokens` is the token count. Claude Code computes both from the same input + cache_creation + cache_read sum (output excluded) and sets them together, so count and percentage agree. A null `used_percentage` renders a dash.

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

**Last reviewed: v2.1.280 (September 23, 2026).** The per-version log, field-change history, and tracked GitHub issues live in `docs/claude-code-version-reviews.md`. When reviewing a newer release, append an entry there and update this line. Verify claims against the installed binary (`~/.local/share/claude/versions/<version>`; the statusline JSON builder is the function containing `exceeds_200k_tokens:`), not only the changelog.

### Statusline-related settings

- `statusLine.refreshInterval` (introduced v2.1.97) — **seconds** (min 1) between idle re-invocations of the statusline command, set inside the `statusLine` object in `~/.claude/settings.json` (verified in the v2.1.280 binary: `Math.max(1, refreshInterval) * 1000`). Useful when displaying time-elapsed metrics that should update without user activity. We don't set this ourselves; users who want idle refresh can opt in. Independently of this, Claude Code re-runs the script when a rate-limit window reaches `resets_at`, when a warm prompt cache reaches `expires_at`, on permission-mode change, and on vim toggle (300ms debounce, in-flight script cancelled).
- `statusLine.padding` — left padding in characters (default 0).
- `statusLine.hideVimModeIndicator` — suppresses the built-in `-- INSERT --` when the script renders `vim.mode` itself.
- The statusline command receives `COLUMNS` and `LINES` env vars (since v2.1.153); `tput cols` does not work because output is captured.

### Available JSON fields not yet used

These exist in the statusline JSON but we don't leverage them:

- `exceeds_200k_tokens` — whether the latest response exceeded 200k tokens. Dropped from display in plugin v2.5.0: no current model has a >200k price tier, and the context color already warns from 150k
- `session_id`, `session_name` (from `--name`/`/rename`), `transcript_path`, `version`
- `prompt_id` — UUID of the user prompt being processed, matches OTel `prompt.id` (since v2.1.196; absent until first user input)
- `workspace.current_dir` — same value as `cwd`; preferred alias in official docs
- `workspace.repo.{host,owner,name}` — repo identity from `origin` remote (since v2.1.145)
- `workspace.added_dirs` — directories added via `/add-dir` (since v2.1.47)
- `vim.mode` — current vim mode (NORMAL/INSERT/VISUAL/VISUAL LINE)
- `rate_limits.spend_limit.{used_percentage,resets_at}` — gateway spend-limit window (since v2.1.251; `used_percentage` may exceed 100)
- `prompt_cache.*` — per-session prompt-cache telemetry (since v2.1.251; absent until the first API response; miss "likely cause" since v2.1.260). `warm`/`hit_ratio` could drive a cache indicator
- `remote.session_id` — present in Remote Control sessions (binary-verified, undocumented)
- `pr.url` — PR link
- `output_style.name` — current output style
- `cost.total_api_duration_ms` — API time vs wall time
- `context_window.remaining_percentage` — inverse of `used_percentage`
- `context_window.current_usage.*` — per-bucket token breakdown (its input + cache sum equals `total_input_tokens`)
- `context_window.total_output_tokens` — output tokens from the most recent API response
- `worktree.{path,branch,original_cwd,original_branch}` — `--worktree` session details
