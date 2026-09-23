# Claude Code version reviews

Log of Claude Code releases reviewed for statusline-relevant changes (stdin JSON fields, context/auto-compact behavior, rate limits, model IDs, statusline invocation). `CLAUDE.md` records the last reviewed version; append new reviews here.

## Per-version log

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

**v2.1.260** — `prompt_cache` gained a "likely cause" for prompt-cache misses (also shown in `/cost`); not displayed. `workspace.repo.owner` now carries the full namespace for GitLab nested subgroups.

**v2.1.261–v2.1.279** — No statusline JSON changes. v2.1.265 fixed `workspace.repo` for GitLab nested projects. v2.1.267 added the `maxEffortLevel` setting (caps selectable effort; `effort.level` still reports the live value). v2.1.269 added `/output-style [name]` (`output_style.name` unchanged). v2.1.271 enabled fast mode in Remote sessions.

**v2.1.280** — **Claude Opus 5.5 released** (`claude-opus-5-5`, now the default Opus, 1M context, $4/$20 per Mtok). Thinking can't be disabled on it (API default effort `medium`). Parser already output "Opus 5.5"; regression tests added in plugin v2.5.0. Because Claude Code reports thinking as on unless the user turns it off, and it can't be turned off on Opus 5.5/Fable, plugin v2.5.0 replaced the near-permanent `*` thinking marker with a `∅` shown only when `thinking.enabled` is false. Docs-verified at the same time: all current 1M models are priced flat (no >200k long-context premium; fast mode is a flat 2x), so plugin v2.5.0 dropped the `>200k` badge; plugins still can't contribute `statusLine` (plugin `settings.json` supports only `agent` and `subagentStatusLine`), so `/simple-statusline:setup` stays.

**Binary-verified in v2.1.280 (not in changelog):** the statusline JSON builder is unchanged from v2.1.259 — same fields and conditions, `thinking.enabled` is `true` unless explicitly disabled, `effort.level` values still include `auto`. `context_window.used_percentage` and `current_usage` are null together before the first response; `total_input_tokens` = input + cache_creation + cache_read of the latest response (0 before it) — plugin v2.5.0 reads it directly instead of summing `current_usage`. `workspace.git_worktree` is set only when the git dir's parent is named `worktrees` (linked worktrees; submodules excluded) — plugin v2.5.0 uses it for `[wt:…]` instead of its own `.git`-is-a-file check, which false-positived on submodules. `model.display_name` = managed model-picker label → registry label → catalog name (+ " (1M context)" for `[1m]` IDs) → raw ID. Auto-compact constants and resolution order unchanged. Skipped/unpublished version numbers in this range: 2.1.262, 2.1.264.

## Statusline JSON field changes in v2.1.29–v2.1.280

v2.1.47 added `workspace.added_dirs`. v2.1.50 introduced the `[1m]` suffix on model IDs for 1M context models (handled in `src/model.go` — we strip `[...]` before version parsing). v2.1.69 added the `worktree` object (name, path, branch, original_cwd, original_branch). v2.1.80 added `rate_limits` with five_hour/seven_day windows. v2.1.97/98 added `workspace.git_worktree` (skipped at first; used for the worktree tag from plugin v2.5.0). v2.1.119 added `effort.level` and `thinking.enabled` (now displayed inline with the model name). v2.1.145 added `pr.{number,url,review_state}` (displayed as a PR badge in the git segment from plugin v2.3.0) and `workspace.repo.{host,owner,name}` (not used). v2.1.196 added `prompt_id` (not used — correlation UUID, not display data). v2.1.234 added `pr.kind` (`"mr"`, rendered as `!N` from plugin v2.4.0). v2.1.251 added `rate_limits.spend_limit` and `prompt_cache` (not displayed). `fast_mode` (boolean) is documented and emitted by v2.1.259 (displayed as `⚡` from plugin v2.4.0). Rate-limit windows are now dropped from the JSON once their `resets_at` passes, so an absent window no longer implies an old Claude Code version. `session_name` and `workspace.current_dir` are also now documented in the official statusline docs. All other fields remained stable. v2.1.260–v2.1.280 added no fields.

## Usage API changes

The OAuth API call to `/api/oauth/usage` has been removed as of plugin v2.1.0. Rate limit data is now sourced exclusively from the native `rate_limits` field in stdin JSON (available since Claude Code v2.1.80). Users on older Claude Code versions will see dashes for rate limit data.

## Open issues to track

- [#22221](https://github.com/anthropics/claude-code/issues/22221) — Expose rate limits in statusline JSON (partially resolved by v2.1.80 which added `rate_limits`; original request also asked for billing cycle/plan-level data which is not yet available)

## Closed without resolution (feature still unavailable)

- [#39420](https://github.com/anthropics/claude-code/issues/39420) — Add permission_mode to statusline JSON. Closed March 2026 as duplicate of #31167, which is itself closed as a duplicate. Permission mode is still not in statusline JSON as of v2.1.280 (the builder receives `permissionMode` but only uses it to pick the plan-mode model ID) (v2.1.203 added a built-in grey ⏸ footer badge for manual mode instead).
- [#37227](https://github.com/anthropics/claude-code/issues/37227) — Expose per-model rate limits in statusline `rate_limits`. Closed May 2026 as inactive/not planned.
- [#33310](https://github.com/anthropics/claude-code/issues/33310) — Expose background task count in statusline JSON. Closed May 2026 as inactive/not planned.

## Resolved issues

- [#38392](https://github.com/anthropics/claude-code/issues/38392) / [#39399](https://github.com/anthropics/claude-code/issues/39399) / [#37764](https://github.com/anthropics/claude-code/issues/37764) / [#37701](https://github.com/anthropics/claude-code/issues/37701) / [#36187](https://github.com/anthropics/claude-code/issues/36187) — Add effort level to statusline JSON. Resolved by Claude Code v2.1.119 (`effort.level` + `thinking.enabled`). Surfaced in this plugin from v2.2.0.
