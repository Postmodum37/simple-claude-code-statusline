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
- `worktree.*` - Worktree metadata (only present in `--worktree` sessions, since v2.1.69)
- `rate_limits.{five_hour,seven_day}` - Rate limit windows (since v2.1.80)
- `effort.level` - Reasoning effort: `low|medium|high|xhigh|max` (since v2.1.119; docs no longer list `auto` — ultracode reports as `xhigh`)
- `thinking.enabled` - Whether extended thinking is on (since v2.1.119)
- `pr.{number,url,review_state,kind}` - Open PR for the current branch (since v2.1.145; absent until a PR is found; `kind: "mr"` for GitLab merge requests)
- `fast_mode` - Whether fast mode is on (boolean, always present in v2.1.259)
- `prompt_cache.*` - Prompt-cache telemetry (`warm`, `ttl`, `expires_at`, `hit_ratio`, …; absent before the first request) — not displayed
- `rate_limits.spend_limit` - Overage spend-limit window, gateway accounts only — not displayed
- `remote.session_id` - Present in Remote Control sessions — not displayed
- `workspace.repo.{host,owner,name}` - Repo identity from the `origin` remote (since v2.1.145)
- `exceeds_200k_tokens` - Whether the most recent API response exceeded 200k tokens

The binary outputs two lines of ANSI-escaped text:
1. Model [⚡fast] [thinking-marker] [•effort] [agent] | Directory | Git branch + status + PR badge | Session lines changed
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

Three screenshots in the repo root are referenced by README.md:

- `screenshot.png` — Main hero image: Opus model, moderate context (~42%), lines changed
- `screenshot-git.png` — Git features: higher context (~65%), git status visible
- `screenshot-sonnet.png` — Sonnet model, lower context (~30%), no lines changed

**How to update:**

1. Pipe mock JSON to the binary to generate ANSI output for each scenario:
   ```sh
   echo '{"model":{"id":"claude-opus-4-6[1m]"},"cwd":"/path/to/repo","workspace":{"project_dir":"/path/to/repo"},"context_window":{"used_percentage":42,"context_window_size":200000,"current_usage":{"input_tokens":76000,"output_tokens":8000}},"cost":{"total_cost_usd":1.25,"total_duration_ms":420000,"total_lines_added":51,"total_lines_removed":14}}' | ./bin/statusline.sh
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
- `git` - Repository status (with `--no-optional-locks` to avoid conflicts)
- `~/.claude/settings.json`, `<project>/.claude/settings.json`, `<project>/.claude/settings.local.json`, managed-settings.json - `autoCompactEnabled` and `autoCompactWindow` (highest-precedence file wins)
- `~/.claude.json` - Legacy `autoCompactEnabled` fallback and the server-provided `autoCompactWindowsCache` per-model window table

## Key Implementation Notes

- Adding a new JSON field requires updating the `StdinData` struct in `src/stdin.go` and this file's "Available JSON fields" section
- Go source is in `src/` with one package (`main`): stdin parsing, model ID parsing, formatting, git status, rate-limit data, auto-compact detection, and ANSI rendering
- Caches to `${CLAUDE_CODE_TMPDIR:-/tmp}/claude-*` (git: 5s TTL). Atomic writes via tmpfile + rename.
- Colors use Tokyo Night palette as constants in `src/render.go`. Effort levels use a separate semantic gradient via `effortColor()`: low=muted, medium=white, high=warn, xhigh=high, max=crit, auto=accent (auto is a mode, not a gradient slot; official docs no longer list `auto` as an `effort.level` value — kept as harmless legacy handling).
- Fast mode (`fast_mode: true`) shows as a yellow `⚡` immediately after the model name; thinking-on shows as a muted `*` after that; effort shows as `•{level}` after the `*` (when present), all before the agent bracket.
- Context color (`contextColor()` in `src/render.go`) is the worse of the percent band (`getSemanticColor()`: ≤50/≤75/≤90) and an absolute-token band (≤150k/≤300k/≤400k). On ≤200k windows this equals the percent band; on 1M windows it turns yellow at 150k, orange at 300k, red at 400k. Rationale: quality degrades with absolute context length (Pocock ~100–150k safe zone; Claude Code team's Thariq Shihipar: rot around 300–400k on 1M). The auto-compact marker is separate and purely mechanical.
- Rate-limit color (`usageColor()`) is pace-based: projected end-of-window usage = used% × window / elapsed, where elapsed is derived from `resets_at` (5h = 18000s, 7d = 604800s) and clamped to ≥5% of the window. Bands: <85 ok, <100 warn, <120 high, else crit; capped at warn below 20% used; floored at high/crit at 75%/90% used. Falls back to percent bands without a reset time.
- Lines changed shows session-cumulative totals from `cost.total_lines_added`/`cost.total_lines_removed`
- Auto-compact indicator `(↻N%)` shown when auto-compact is enabled. `GetCompactThreshold()` in `src/compact.go` replicates Claude Code's resolution (verified against the v2.1.259 binary): kill switches `DISABLE_AUTO_COMPACT`/`DISABLE_COMPACT`; `autoCompactEnabled` from settings.json (managed > local > project > user) then legacy `~/.claude.json`; window = min(model window, `CLAUDE_CODE_AUTO_COMPACT_WINDOW` (clamped 100k–1M) → `autoCompactWindow` setting (set by `/autocompact` or `--autocompact`) → `autoCompactWindowsCache[model]` in `~/.claude.json`); effective = window − min(`CLAUDE_CODE_MAX_OUTPUT_TOKENS`, 20000); threshold = effective − 13000, lowered (never raised) by `CLAUDE_AUTOCOMPACT_PCT_OVERRIDE`. Not replicated: server-side statsig overrides and the `d1()`/experiment 200k caps for some 1M models — the JSON exposes no effective-window field.
- `>200k` indicator driven by the native `exceeds_200k_tokens` boolean from stdin (fast mode pricing threshold)
- PR badge (`prBadge()` in `src/render.go`) renders `#<number>` (or `!<number>` when `pr.kind == "mr"`, matching Claude Code's own GitLab footer) plus a review-state glyph (✓ approved / ⏳ pending / ✗ changes_requested / ◌ draft) inside the git segment; requires git info to be present
- Context display uses `used_percentage` as single source of truth for bar/percentage. `current_usage.*` drives the absolute token count, summed as input + cache_creation + cache_read (output tokens excluded) — this is exactly the sum Claude Code uses for `used_percentage` and `total_input_tokens` (verified in the v2.1.259 binary), so count and percentage agree. (Note: prior to Claude Code v2.1.132 `current_usage` reported cumulative session totals — that bug is now fixed and the field is trustworthy.)

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

### Last reviewed: v2.1.259 (September 3, 2026)

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

**v2.1.64–v2.1.68** — Opus 4/4.1 removed (users auto-migrated to Opus 4.6). Effort defaults to medium for Opus 4.6 on Max/Team. No statusline field changes.

**v2.1.69** — `worktree` object added to statusline JSON (`name`, `path`, `branch`, `original_cwd`, `original_branch`). Present only during `--worktree` sessions. Effort display added to Claude Code's own logo/spinner.

**v2.1.70–v2.1.76** — `/effort` command added (v2.1.76). Effort levels simplified to low/medium/high (v2.1.72). 1M context for Opus 4.6 default for Max/Team/Enterprise (v2.1.75). `PostCompact` hook added (v2.1.76). No new statusline JSON fields.

**v2.1.77–v2.1.79** — Opus 4.6 output tokens increased to 64k (v2.1.77). Status line vim mode toggle fix (v2.1.77). Status line workspace trust fix (v2.1.79). `StopFailure` hook (v2.1.78). No new statusline JSON fields.

**v2.1.80** — **`rate_limits` field added to statusline JSON** with `five_hour` and `seven_day` windows, each containing `used_percentage` (0-100) and `resets_at` (Unix epoch seconds). Only present for Claude.ai Pro/Max subscribers after the first API response. This eliminated the need for our OAuth API call.

**v2.1.81–v2.1.83** — `CwdChanged`/`FileChanged` hooks (v2.1.83). Transcript search (v2.1.83). No statusline field changes.

**v2.1.84** — Token counts >=1M display as "1.5m" in Claude Code's UI (our formatting already handles this). `${CLAUDE_PLUGIN_DATA}` variable for persistent plugin state. Plugin `userConfig` options with keychain storage. No statusline field changes.

**v2.1.85** — Conditional `if` field for hooks (reduces process spawning). No statusline field changes.

**v2.1.86** — Fixed "statusline showing other session's model" — the statusline previously received stale model data from parallel sessions. No JSON shape change; reliability improvement only.

**v2.1.87–v2.1.96** — Bug fixes only across this run (Cowork dispatch, hooks, voice, memory, MCP persistence, Bedrock setup, `/cost` cache breakdown, Linux sandbox, Bedrock 403 regression). No statusline JSON changes.

**v2.1.97–v2.1.98** — **`workspace.git_worktree` added to statusline JSON** (string|null path of the active git worktree, null when in main worktree). Also added `refreshInterval` *setting* (configurable idle refresh cadence — see "Statusline-related settings" below). We evaluated `workspace.git_worktree` and skipped it: our existing logic prefers the rich `worktree` object, then falls back to git-detected worktree from `git rev-parse`, which catches the same cases.

**v2.1.101–v2.1.110** — Bug fixes and feature additions unrelated to statusline (`/team-onboarding`, CA certs, `/tui`, push notifications, Remote Control, prompt caching TTL flags, `EnterWorktree` path param, PreCompact hook). No statusline JSON changes.

**v2.1.111** — **Claude Opus 4.7 released** (`claude-opus-4-7`). Model ID parsing handles this correctly (substring match on "opus" outputs "Opus 4.7"). Also introduced `xhigh` effort tier between `high` and `max`.

**v2.1.112–v2.1.118** — Bug fixes, native binary, sandbox networking, vim, security fixes, `/usage` tab merge of `/cost`+`/stats`, agent frontmatter hooks. v2.1.117 changed default effort for Pro/Max on Opus/Sonnet 4.6 to `high`. No statusline JSON changes.

**v2.1.119** — **`effort.level` and `thinking.enabled` added to statusline JSON.** Resolves issues #38392/#39399/#37764/#37701/#36187 (5 duplicate requests). Shape: `effort.level: "low"|"medium"|"high"|"xhigh"|"max"|"auto"`, `thinking.enabled: bool`. Reflects live session state including `/effort` overrides. Hooks `PostToolUse`/`PostToolUseFailure` gained `duration_ms` (not statusline stdin).

**v2.1.121** — Stabilization of `effort.level`/`thinking.enabled` ship. OTel telemetry gained `stop_reason`, `finish_reasons`, `user_system_prompt` attributes (not statusline stdin).

**v2.1.122–v2.1.131** — Bedrock service tier env var, OTel numeric attributes, OAuth fix, model gateway, `claude project purge`, Windows PowerShell detection, plugin `--plugin-dir`/`--plugin-url` flags, `skillOverrides`, gateway model discovery opt-in, VS Code/Mantle fixes. v2.1.128 fixed "1M-context models falsely blocked with 'Prompt is too long'" and collapsed Opus 4.7 picker duplicates. No statusline JSON changes.

**v2.1.132** — **Critical bug fix:** `context_window.current_usage.{input_tokens,output_tokens}` no longer report cumulative session totals — the values now accurately reflect the current in-flight context window and match `/context` output. We never applied a correction factor, so this fix is invisible to our code, but the field is now trustworthy. Also: `CLAUDE_CODE_SESSION_ID` env var is now set in Bash subprocess environments (not a statusline stdin field).

**v2.1.133** — `effort.level` documented as a hook JSON input field, plus `$CLAUDE_EFFORT` env var for hooks/Bash. Also `worktree.baseRef` setting (`fresh`|`head`) controlling worktree base branch. No new statusline fields (we already consumed `effort.level` since v2.1.119).

**v2.1.134–v2.1.140** — No statusline-relevant changes (`claude agents` view, `/goal` command, auto-mode classifier rules, bug fixes).

**v2.1.141** — Fixed multi-line statusline output dropping/corrupting rows when any line exceeds terminal width. Rendering fix only; no JSON changes.

**v2.1.142** — Fast mode now defaults to Opus 4.7 (was Opus 4.6) — `model.id` in fast-mode sessions changed accordingly.

**v2.1.143–v2.1.144** — `/model` is now session-scoped (press `d` in picker for default), so `model.id` reliably reflects the session model. No JSON shape changes.

**v2.1.145** — **`pr` object added to statusline JSON** (`number`, `url`, `review_state: approved|pending|changes_requested|draft`) — open PR for the current branch, absent until found and removed on merge/close; `review_state` may be independently absent. **`workspace.repo` added** (`host`, `owner`, `name` from the `origin` remote). Also `claude agents --json` for external status bars. PR badge displayed by this plugin from v2.3.0.

**v2.1.147–v2.1.152** — No statusline JSON changes. v2.1.149 fixed `effort.level` to reflect skill/agent `effort:` frontmatter instead of the user's baseline `/effort` setting — treat the field as dynamic per-turn.

**v2.1.153** — Statusline commands now receive `COLUMNS` and `LINES` env vars for terminal-width-aware output. Not used by us yet.

**v2.1.154** — **Claude Opus 4.8 released** (`claude-opus-4-8`), defaults to high effort; `/effort xhigh` promoted for hardest tasks (so `xhigh` is now a common `effort.level` value — already rendered and colored by `effortColor()`). Fast mode moved to Opus 4.8 at 2x rate; `CLAUDE_CODE_OPUS_4_6_FAST_MODE_OVERRIDE` deprecated. Model ID parsing handles 4.8 correctly ("Opus 4.8").

**v2.1.155–v2.1.165** — No statusline changes. Notable: auto mode on Bedrock/Vertex/Foundry for Opus 4.7/4.8 (v2.1.158), `/effort` now confirms persistence as default for new sessions (v2.1.162).

**v2.1.166** — `MAX_THINKING_TOKENS=0`, `--thinking disabled`, and per-model thinking toggle now disable thinking on models that think by default — `thinking.enabled` can now be `false` on such models.

**v2.1.167–v2.1.169** — v2.1.169 fixed footer hints (e.g. "esc to interrupt") not showing for users with a custom statusline. No JSON changes.

**v2.1.170** — **Claude Fable 5 released** (`claude-fable-5`, Mythos-class tier above Opus; appears as `claude-fable-5[1m]` with 1M context). Family parsing for "fable" added to `src/model.go` in plugin v2.3.0 (outputs "Fable 5"); previously it fell through to the displayName fallback and showed just "Fable".

**v2.1.172–v2.1.175** — No statusline JSON changes. v2.1.172 fixed doubled `[1M][1m]` model-ID suffixes (our parser strips at the first `[`, so it was already tolerant). v2.1.173 fixed Fable 5 `[1m]` normalization — `model.id` for Fable may now arrive without the suffix (handled either way).

**v2.1.176–v2.1.178** — No statusline JSON changes. v2.1.176 added the `footerLinksRegexes` setting (regex-matched link badges in the footer row — adjacent to but separate from the statusline). v2.1.178 fixed statusline OSC 8 links with custom URI schemes (e.g. `vscode://`) not opening in `claude agents` — relevant only if we ever emit clickable links.

**v2.1.179–v2.1.195** — No statusline JSON changes across this run (auto-mode safety, MCP reliability, background agents, voice, sandbox settings). v2.1.183 fixed fullscreen TUI corruption that could render the statusline mid-screen in Windows Terminal.

**v2.1.196** — **`prompt_id` added to statusline JSON**: UUID of the user prompt currently being processed, matching the `prompt.id` OTel attribute for event correlation. Absent until the first user input. Not useful for display; not adopted.

**v2.1.197** — **Claude Sonnet 5 released** (`claude-sonnet-5`, native 1M context, now the default model in Claude Code). Verified: existing parsing in `src/model.go` handles the single-digit version (outputs "Sonnet 5", `[1m]` suffix stripped); regression tests added in `src/model_test.go`.

**v2.1.198–v2.1.207** — No statusline JSON changes. v2.1.203 added a grey ⏸ footer badge for manual permission mode — permission mode is still NOT exposed in statusline JSON (see closed issue #39420 below). v2.1.198: the built-in Explore agent now inherits the main session's model.

**v2.1.208** — Fixed the context window (and auto-compact indicator) briefly resetting to 200k after CLI auto-updates, which caused a false "100% context used" on long-context sessions. Reliability fix for data we display; no JSON shape change.

**v2.1.209–v2.1.211** — No statusline JSON changes. v2.1.210 fixed `/clear` not resetting the session cost counter — `cost.total_cost_usd` now starts at $0 after `/clear`.

**v2.1.212–v2.1.215** — No statusline JSON changes. v2.1.214 added reasoning effort to the separate `subagentStatusLine` payload (agent-panel hook, not our stdin).

**v2.1.216** — Fixed the statusline running twice on resume. No JSON changes.

**v2.1.217–v2.1.218** — No statusline JSON changes. v2.1.217 fixed auto-compact never triggering for Opus 4.8 on Bedrock; footer PR badge links now clickable without hyperlink detection (`FORCE_HYPERLINK=0` opts out).

**v2.1.219** — **Claude Opus 5 released** (`claude-opus-5`, now the default Opus, 1M context, fast mode at $10/$50 per Mtok). Opus 4.7 removed from fast mode; `/fast` applies to Opus 5 and Opus 4.8. Parser outputs "Opus 5" (regression test added in plugin v2.4.0).

**v2.1.220–v2.1.222** — No statusline JSON changes. v2.1.221 introduced **`/autocompact <tokens|auto>` and `--autocompact`**, which write the `autoCompactWindow` setting (100k–1M) — this moves the auto-compact marker; consumed by `src/compact.go` from plugin v2.4.0. v2.1.221 also fixed the thinking toggle having no effect mid-session (`thinking.enabled` reliable after toggles). v2.1.222 fixed sessions not linking to PRs created after push.

**v2.1.223** — **Auto-compact window behavior change.** `CLAUDE_CODE_DISABLE_1M_CONTEXT=1` now holds every native-1M model to 200K via auto-compaction; unrecognized model IDs are held within the assumed window (`CLAUDE_CODE_DISABLE_UNKNOWN_MODEL_WINDOW_ENFORCEMENT=1` restores old behavior). Neither is visible in statusline JSON, so the marker can be off for those users.

**v2.1.224–v2.1.233** — No statusline JSON changes. v2.1.225 added gateway spend-limit warnings (precursor to `rate_limits.spend_limit`). v2.1.229: "prompt is too long" errors explain why auto-compaction couldn't recover. v2.1.233 first names **Mythos 5** as a model family (todo tools removed on Opus 4.8/Sonnet 5/Fable 5/Mythos 5 and newer). The v2.1.259 binary contains `claude-mythos-5` and `claude-mythos-5-1` IDs with display names "Mythos 5"/"Mythos 5.1" — family parsing for "mythos" added in plugin v2.4.0.

**v2.1.234** — **`pr.kind` added to statusline JSON**: GitLab merge requests (GitLab remote + authenticated `glab`) now populate `pr` with `kind: "mr"` (absent for GitHub PRs); `review_state` is `approved` when mergeable, `pending` otherwise, `draft` for drafts (`changes_requested` never occurs for MRs). Claude Code's footer renders MRs as `!N`; the plugin does the same from v2.4.0. Also: Remote Control effort picks apply to terminal sessions, so `effort.level` can change from a phone.

**v2.1.235–v2.1.238** — No statusline JSON changes. v2.1.237 added the built-in "Concise" output style (new `output_style.name` value). v2.1.236 added `ANTHROPIC_DEFAULT_MODEL`.

**v2.1.239** — `cost.total_cost_usd` (and `/cost`) now includes the 1.1× US-only-inference premium for data-residency workspaces. No JSON shape change.

**v2.1.240–v2.1.243** — v2.1.243: **`modelPricing` managed setting** — `cost.total_cost_usd` may reflect contracted per-model rates instead of list price (requires v2.1.242). **Fixed statusline `rate_limits` fields showing a window's pre-reset usage after the window reset while idle.** Also `promptCacheTtl` settings, `/tasks` shows model + effort per subagent.

**v2.1.245–v2.1.246** — v2.1.246 fixed the statusline's cost and duration resetting to zero after visiting the agents view. No JSON changes.

**v2.1.247** — **Sonnet 5's default auto-compact window changed to its full 1M context**, so 1M sessions compact at ~967K (was ~934K). 967K = 1M − 20k reserve − 13k buffer, which independently confirms our formula.

**v2.1.248–v2.1.250** — No statusline JSON changes. v2.1.248: footer PR badge polls GitHub less often; `[1m]` suffixes render literally in `/model`.

**v2.1.251** — **Two new statusline JSON fields:** `rate_limits.spend_limit.{used_percentage,resets_at}` (Claude apps gateway spend limits only; `used_percentage` may exceed 100) and a `prompt_cache` object (`warm`, `caching_observed`, `ttl` "5m"|"1h", `expires_at`, `requests`, `misses`, `expected_rebuilds`, `hit_ratio`, `cache_write_tokens`, `miss_recache_tokens`, `last_miss_at`, `recache_tokens_if_cold`; main conversation only, absent until the first API response). Parsed but not displayed. Also `PreModelSwitch`/`PostModelSwitch` hooks; `/effort` saves a default per model; Opus 5 with thinking off sends effort as `high` when xhigh/max was chosen; project-level `env` can no longer set `CLAUDE_CODE_TMPDIR` (our cache-dir env var).

**v2.1.252** — No statusline changes.

**v2.1.257** — **Claude Fable 5.1 released** (`claude-fable-5-1`, now the default Fable model, 1M context; the `fable` alias resolves to 5.1 from v2.1.255 and saved `claude-fable-5[1m]` settings are rewritten to the alias). Parser outputs "Fable 5.1" (regression test added in plugin v2.4.0). Also `s` in `/effort` for session-only effort; `--effort` lifts a new model's default-effort hold for the session only.

**v2.1.258–v2.1.259** — No statusline JSON changes. v2.1.259 fixed concurrent sessions silently reverting each other's `~/.claude.json` changes (we read `~/.claude.json` for legacy `autoCompactEnabled` and `autoCompactWindowsCache`) and fixed `CLAUDE_CODE_MAX_CONTEXT_TOKENS` being ignored for Vertex-style `@YYYYMMDD` IDs.

**Binary-verified in v2.1.259 (not in changelog):** the statusline JSON builder also emits a top-level **`fast_mode`** boolean (documented, undated; displayed as `⚡` from plugin v2.4.0) and `remote: {session_id}` in Remote Control sessions. `thinking` is always present; `effort` only when the model supports the effort parameter. `used_percentage` = round((input + cache_creation + cache_read) / context_window_size × 100), clamped, excluding output tokens. Claude Code's own context indicator warns 20k tokens before the compact threshold and blocks 3k tokens before the effective window. Skipped/unpublished version numbers in this range: 2.1.213, 2.1.230, 2.1.242, 2.1.244, 2.1.249, 2.1.253–2.1.256.

### Statusline JSON field changes in v2.1.29–v2.1.259

v2.1.47 added `workspace.added_dirs`. v2.1.50 introduced the `[1m]` suffix on model IDs for 1M context models (handled in `src/model.go` — we strip `[...]` before version parsing). v2.1.69 added the `worktree` object (name, path, branch, original_cwd, original_branch). v2.1.80 added `rate_limits` with five_hour/seven_day windows. v2.1.97/98 added `workspace.git_worktree` (skipped — redundant with our existing worktree handling). v2.1.119 added `effort.level` and `thinking.enabled` (now displayed inline with the model name). v2.1.145 added `pr.{number,url,review_state}` (displayed as a PR badge in the git segment from plugin v2.3.0) and `workspace.repo.{host,owner,name}` (not used). v2.1.196 added `prompt_id` (not used — correlation UUID, not display data). v2.1.234 added `pr.kind` (`"mr"`, rendered as `!N` from plugin v2.4.0). v2.1.251 added `rate_limits.spend_limit` and `prompt_cache` (not displayed). `fast_mode` (boolean) is documented and emitted by v2.1.259 (displayed as `⚡` from plugin v2.4.0). Rate-limit windows are now dropped from the JSON once their `resets_at` passes, so an absent window no longer implies an old Claude Code version. `session_name` and `workspace.current_dir` are also now documented in the official statusline docs. All other fields remained stable.

### Statusline-related settings

- `statusLine.refreshInterval` (introduced v2.1.97) — **seconds** (min 1) between idle re-invocations of the statusline command, set inside the `statusLine` object in `~/.claude/settings.json` (verified in the v2.1.259 binary: `Math.max(1, refreshInterval) * 1000`). Useful when displaying time-elapsed metrics that should update without user activity. We don't set this ourselves; users who want idle refresh can opt in. Independently of this, Claude Code re-runs the script when a rate-limit window reaches `resets_at`, when a warm prompt cache reaches `expires_at`, on permission-mode change, and on vim toggle (300ms debounce, in-flight script cancelled).
- `statusLine.padding` — left padding in characters (default 0).
- `statusLine.hideVimModeIndicator` — suppresses the built-in `-- INSERT --` when the script renders `vim.mode` itself.
- The statusline command receives `COLUMNS` and `LINES` env vars (since v2.1.153); `tput cols` does not work because output is captured.

### Usage API changes

The OAuth API call to `/api/oauth/usage` has been removed as of plugin v2.1.0. Rate limit data is now sourced exclusively from the native `rate_limits` field in stdin JSON (available since Claude Code v2.1.80). Users on older Claude Code versions will see dashes for rate limit data.

### Available JSON fields not yet used

These exist in the statusline JSON but we don't leverage them:

- `version` — Claude Code version string (e.g., "2.1.211")
- `session_name` — custom session name from `--name`/`/rename` (absent if unset)
- `prompt_id` — UUID of the user prompt being processed, matches OTel `prompt.id` (since v2.1.196; absent until first user input)
- `workspace.current_dir` — same value as `cwd`; preferred alias in official docs
- `workspace.repo.{host,owner,name}` — repo identity from `origin` remote (since v2.1.145)
- `vim.mode` — current vim mode (NORMAL/INSERT/VISUAL/VISUAL LINE)
- `rate_limits.spend_limit.{used_percentage,resets_at}` — gateway spend-limit window (since v2.1.251; `used_percentage` may exceed 100)
- `prompt_cache.*` — per-session prompt-cache telemetry (since v2.1.251; absent until the first API response). `warm`/`hit_ratio` could drive a cache indicator
- `remote.session_id` — present in Remote Control sessions (binary-verified, undocumented)
- `pr.url` / `pr.kind` beyond the `!`/`#` prefix
- `output_style.name` — current output style
- `cost.total_api_duration_ms` — API time vs wall time
- `context_window.remaining_percentage` — pre-calculated remaining % (inverse of `used_percentage`)
- `transcript_path` — path to conversation transcript file
- `context_window.total_input_tokens` — tokens currently in the context window (input incl. cache reads/writes), from the most recent API response; cumulative before Claude Code v2.1.132
- `context_window.total_output_tokens` — output tokens from the most recent API response; cumulative before Claude Code v2.1.132
- `workspace.added_dirs` — directories added via `/add-dir` (since v2.1.47)
- `workspace.git_worktree` — name of the linked git worktree when cwd is inside one; populated for any `git worktree add` worktree, unlike `worktree.*` which is `--worktree`-sessions-only (since v2.1.97; docs now describe it as a name, absent in the main working tree). Evaluated and skipped — redundant with our existing `worktree` object handling and git-detection fallback in `src/git.go`.
- `worktree.path` — absolute path to worktree directory
- `worktree.original_cwd` — directory before entering worktree
- `worktree.original_branch` — git branch before entering worktree

### Open issues to track

- [#22221](https://github.com/anthropics/claude-code/issues/22221) — Expose rate limits in statusline JSON (partially resolved by v2.1.80 which added `rate_limits`; original request also asked for billing cycle/plan-level data which is not yet available)

### Closed without resolution (feature still unavailable)

- [#39420](https://github.com/anthropics/claude-code/issues/39420) — Add permission_mode to statusline JSON. Closed March 2026 as duplicate of #31167, which is itself closed as a duplicate. Permission mode is still not in statusline JSON as of v2.1.259 (the v2.1.259 builder receives `permissionMode` but only uses it to pick the plan-mode model ID) (v2.1.203 added a built-in grey ⏸ footer badge for manual mode instead).
- [#37227](https://github.com/anthropics/claude-code/issues/37227) — Expose per-model rate limits in statusline `rate_limits`. Closed May 2026 as inactive/not planned.
- [#33310](https://github.com/anthropics/claude-code/issues/33310) — Expose background task count in statusline JSON. Closed May 2026 as inactive/not planned.

### Resolved issues

- [#38392](https://github.com/anthropics/claude-code/issues/38392) / [#39399](https://github.com/anthropics/claude-code/issues/39399) / [#37764](https://github.com/anthropics/claude-code/issues/37764) / [#37701](https://github.com/anthropics/claude-code/issues/37701) / [#36187](https://github.com/anthropics/claude-code/issues/36187) — Add effort level to statusline JSON. Resolved by Claude Code v2.1.119 (`effort.level` + `thinking.enabled`). Surfaced in this plugin from v2.2.0.
