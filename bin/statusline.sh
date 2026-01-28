#!/bin/bash

# Claude Code Statusline V2 - Tokyo Night Style
# Features: Usage stats, full git status, adaptive collapse

# --- Tokyo Night Color Palette ---
C_RESET="\033[0m"
C_ACCENT="\033[38;5;111m"     # #769ff0 - bright blue
C_MUTED="\033[38;5;146m"      # #a3aed2 - muted text
C_WHITE="\033[38;5;254m"      # #e3e5e5 - white text

# Semantic colors for utilization/context
C_OK="\033[38;5;114m"         # Green (0-50%)
C_WARN="\033[38;5;214m"       # Yellow (51-75%)
C_HIGH="\033[38;5;208m"       # Orange (76-90%)
C_CRIT="\033[38;5;196m"       # Red (91%+)

# Git status colors
C_GIT_ADD="\033[38;5;114m"    # Green for added
C_GIT_MOD="\033[38;5;214m"    # Yellow for modified
C_GIT_DEL="\033[38;5;196m"    # Red for deleted
C_GIT_AHEAD="\033[38;5;81m"   # Cyan for ahead
C_GIT_BEHIND="\033[38;5;208m" # Orange for behind

# PR status colors (matching Claude Code's prompt footer)
C_PR_APPROVED="\033[38;5;114m"   # Green for approved
C_PR_PENDING="\033[38;5;214m"    # Yellow for pending
C_PR_CHANGES="\033[38;5;196m"    # Red for changes requested
C_PR_DRAFT="\033[38;5;146m"      # Muted for draft

# --- Cache current timestamp (avoid multiple date calls) ---
NOW=$(date +%s)

# --- Platform detection (cached for reuse) ---
IS_MACOS=$([[ "$(uname)" == "Darwin" ]] && echo true || echo false)

if [[ "$IS_MACOS" == "true" ]]; then
  stat_mtime() { stat -f %m "$1" 2>/dev/null || echo 0; }
  parse_iso_date() {
    local iso="${1%%.*}"
    date -j -u -f "%Y-%m-%dT%H:%M:%S" "$iso" +%s 2>/dev/null || echo "$NOW"
  }
else
  stat_mtime() { stat -c %Y "$1" 2>/dev/null || echo 0; }
  parse_iso_date() {
    local iso="${1%%.*}"
    date -u -d "${iso//T/ }" +%s 2>/dev/null || echo "$NOW"
  }
fi

# --- Read JSON input ---
input=$(cat)

# --- Cache files (respect CLAUDE_CODE_TMPDIR if set) ---
cache_dir="${CLAUDE_CODE_TMPDIR:-/tmp}"
context_cache="${cache_dir}/claude-context-cache"
git_cache="${cache_dir}/claude-git-cache"
usage_cache="${cache_dir}/claude-usage-cache"
pr_cache="${cache_dir}/claude-pr-cache"

# --- Extract all fields with single jq call ---
# Initialize all variables with defaults first
model_id=""
model_display=""
project_dir=""
cwd=""
context_size=200000
used_pct=0
session_id=""
current_usage=0
lines_added=0
lines_removed=0
duration_ms=0
total_cost=0

eval "$(echo "$input" | jq -r '
  @sh "model_id=\(.model.id // "")",
  @sh "model_display=\(.model.display_name // "")",
  @sh "project_dir=\(.workspace.project_dir // "")",
  @sh "cwd=\(.cwd // "")",
  @sh "context_size=\(.context_window.context_window_size // 200000)",
  @sh "used_pct=\(.context_window.used_percentage // "")",
  @sh "session_id=\(.session_id // "")",
  @sh "lines_added=\(.cost.total_lines_added // 0)",
  @sh "lines_removed=\(.cost.total_lines_removed // 0)",
  @sh "duration_ms=\(.cost.total_duration_ms // 0)",
  @sh "total_cost=\(.cost.total_cost_usd // 0)",
  @sh "current_usage=\(
    (.context_window.current_usage.input_tokens // 0) +
    (.context_window.current_usage.output_tokens // 0) +
    (.context_window.current_usage.cache_creation_input_tokens // 0) +
    (.context_window.current_usage.cache_read_input_tokens // 0)
  )"
' 2>/dev/null)"

# --- Use cached context if current parse returned zero AND same session ---
if [[ $current_usage -eq 0 && -f "$context_cache" ]]; then
  source "$context_cache" 2>/dev/null
  # Only use cache if session matches; otherwise clear to trigger "no data" state
  [[ "$cached_session_id" != "$session_id" ]] && { current_usage=0; used_pct=""; }
elif [[ $current_usage -gt 0 ]]; then
  echo "cached_session_id='$session_id'; current_usage=$current_usage; context_size=$context_size; used_pct=$used_pct" > "$context_cache"
fi

# --- Helper Functions ---

get_semantic_color() {
  local pct=${1:-0}
  # Handle empty or non-numeric input
  [[ ! "$pct" =~ ^[0-9]+$ ]] && pct=0
  if [[ $pct -le 50 ]]; then
    echo "$C_OK"
  elif [[ $pct -le 75 ]]; then
    echo "$C_WARN"
  elif [[ $pct -le 90 ]]; then
    echo "$C_HIGH"
  else
    echo "$C_CRIT"
  fi
}

format_tokens() {
  local tokens=$1
  if [[ $tokens -ge 1000000 ]]; then
    echo "$((tokens / 1000000))m"
  elif [[ $tokens -ge 1000 ]]; then
    echo "$((tokens / 1000))k"
  else
    echo "$tokens"
  fi
}

format_duration() {
  local secs=$1
  # Handle negative duration (clock skew)
  [[ $secs -lt 0 ]] && secs=0
  local mins=$((secs / 60))
  local hours=$((mins / 60))
  local days=$((hours / 24))

  if [[ $days -gt 0 ]]; then
    echo "${days}d$((hours % 24))h"
  elif [[ $hours -gt 0 ]]; then
    echo "${hours}h$((mins % 60))m"
  else
    echo "${mins}m"
  fi
}

format_reset_time() {
  local reset_iso=$1
  [[ -z "$reset_iso" ]] && { echo "0m"; return; }
  local reset_ts=$(parse_iso_date "$reset_iso")
  local diff=$((reset_ts - NOW))

  if [[ $diff -lt 0 ]]; then
    echo "0m"
  elif [[ $diff -lt 3600 ]]; then
    echo "$((diff / 60))m"
  elif [[ $diff -lt 86400 ]]; then
    local hours=$((diff / 3600))
    local mins=$(((diff % 3600) / 60))
    [[ $mins -gt 0 ]] && echo "${hours}h${mins}m" || echo "${hours}h"
  else
    local days=$((diff / 86400))
    local hours=$(((diff % 86400) / 3600))
    [[ $hours -gt 0 ]] && echo "${days}d${hours}h" || echo "${days}d"
  fi
}

# Build progress bar using pure bash (no subshells)
build_bar() {
  local filled=$1
  local empty=$2
  local bar=""
  local i
  for ((i=0; i<filled; i++)); do bar+="▰"; done
  for ((i=0; i<empty; i++)); do bar+="▱"; done
  echo "$bar"
}

# --- Model Display (bash 3.2 compatible - no BASH_REMATCH) ---
model_short=""
case "$model_id" in
  *opus*)   model_short="Opus" ;;
  *sonnet*) model_short="Sonnet" ;;
  *haiku*)  model_short="Haiku" ;;
  *)        model_short="${model_display%% *}" ;;
esac

# Extract version using parameter expansion (bash 3.2 compatible)
# New format: claude-opus-4-5-20251101 -> "4.5", claude-sonnet-4-20250514 -> "4"
# Old format: claude-3-5-sonnet-20241022 -> "3.5"
version_extracted=""

# Check for old format first: claude-{major}-{minor}-{model}-{date}
old_format_part="${model_id#claude-}"  # "3-5-sonnet-20241022" or "opus-4-5-20251101"
if [[ "$old_format_part" =~ ^[0-9]+-[0-9]+- ]]; then
  # Old format detected
  major="${old_format_part%%-*}"
  minor_rest="${old_format_part#*-}"
  minor="${minor_rest%%-*}"
  version_extracted="$major.$minor"
else
  # New format: claude-{model}-{major}[-{minor}]-{date}
  version_part="${model_id#*-*-}"  # Remove "claude-model-": "4-5-20251101" or "4-20250514"
  if [[ "$version_part" =~ ^[0-9]+ ]]; then
    major="${version_part%%-*}"
    minor_rest="${version_part#*-}"
    minor_candidate="${minor_rest%%-*}"
    # Check if next segment is a version number (1-2 digits) not a date (8 digits)
    if [[ "$minor_candidate" =~ ^[0-9]{1,2}$ ]]; then
      version_extracted="$major.$minor_candidate"
    elif [[ -n "$major" ]]; then
      version_extracted="$major"
    fi
  fi
fi

[[ -n "$version_extracted" ]] && model_short="$model_short $version_extracted"

# --- Directory Display (no subshells) ---
dir_display=""
if [[ -n "$cwd" ]]; then
  dir_display="${cwd/#$HOME/~}"

  # Count segments using parameter expansion
  local_path="$dir_display"
  segment_count=0
  while [[ "$local_path" == */* ]]; do
    local_path="${local_path#*/}"
    ((segment_count++))
  done
  ((segment_count++))

  if [[ $segment_count -gt 3 ]]; then
    # Get last 3 segments safely
    oldIFS="$IFS"
    IFS='/' read -ra parts <<< "$dir_display"
    IFS="$oldIFS"
    len=${#parts[@]}
    if [[ $len -ge 3 ]]; then
      dir_display="…/${parts[$((len-3))]}/${parts[$((len-2))]}/${parts[$((len-1))]}"
    fi
  fi
fi

# --- Git Display (cached for 5 seconds) ---
git_branch=""
git_status=""
added=0
modified=0
deleted=0
ahead=0
behind=0

if [[ -n "$project_dir" ]]; then
  git_cache_age=999
  if [[ -f "$git_cache" ]]; then
    git_cache_age=$(( NOW - $(stat_mtime "$git_cache") ))
  fi

  if [[ $git_cache_age -gt 5 ]]; then
    # Refresh git cache
    if git --no-optional-locks -C "$project_dir" rev-parse --git-dir &>/dev/null; then
      git_branch=$(git --no-optional-locks -C "$project_dir" rev-parse --abbrev-ref HEAD 2>/dev/null)
      [[ ${#git_branch} -gt 20 ]] && git_branch="${git_branch:0:17}…"

      git_porcelain=$(git --no-optional-locks -C "$project_dir" status --porcelain 2>/dev/null)

      # Only parse if there's content
      if [[ -n "$git_porcelain" ]]; then
        while IFS= read -r line; do
          [[ -z "$line" ]] && continue
          status="${line:0:2}"
          case "$status" in
            # Untracked files
            "??") ((added++)) ;;
            # Added (staged)
            "A "|"AM"|"AD") ((added++)) ;;
            # Modified (various combinations)
            " M"|"M "|"MM"|"RM"|"CM") ((modified++)) ;;
            # Deleted
            " D"|"D "|"MD"|"RD"|"CD") ((deleted++)) ;;
            # Renamed/Copied (count as added if target is new)
            "R "|"C ") ((added++)) ;;
            # Unmerged/conflict states (show as modified)
            "UU"|"AA"|"DD"|"AU"|"UA"|"DU"|"UD") ((modified++)) ;;
          esac
        done <<< "$git_porcelain"
      fi

      ahead=$(git --no-optional-locks -C "$project_dir" rev-list --count @{u}..HEAD 2>/dev/null || echo 0)
      behind=$(git --no-optional-locks -C "$project_dir" rev-list --count HEAD..@{u} 2>/dev/null || echo 0)

      # Save to cache
      echo "git_branch='$git_branch'; added=$added; modified=$modified; deleted=$deleted; ahead=$ahead; behind=$behind" > "$git_cache"
    fi
  else
    # Use cached git data
    source "$git_cache" 2>/dev/null
  fi

  # Build status string
  [[ $added -gt 0 ]] && git_status+=" ${C_GIT_ADD}✚${added}${C_RESET}"
  [[ $modified -gt 0 ]] && git_status+=" ${C_GIT_MOD}●${modified}${C_RESET}"
  [[ $deleted -gt 0 ]] && git_status+=" ${C_GIT_DEL}✖${deleted}${C_RESET}"
  [[ $ahead -gt 0 ]] && git_status+=" ${C_GIT_AHEAD}↑${ahead}${C_RESET}"
  [[ $behind -gt 0 ]] && git_status+=" ${C_GIT_BEHIND}↓${behind}${C_RESET}"
fi

# --- PR Status (cached 30s) ---
pr_status=""
pr_display=""

if [[ -n "$project_dir" && -n "$git_branch" ]]; then
  pr_cache_age=999
  if [[ -f "$pr_cache" ]]; then
    pr_cache_age=$(( NOW - $(stat_mtime "$pr_cache") ))
  fi

  # Refresh PR cache every 30 seconds
  if [[ $pr_cache_age -gt 30 ]]; then
    # Check for gh CLI and get PR info for current branch
    if command -v gh &>/dev/null; then
      pr_json=$(gh pr view --json state,reviewDecision,isDraft 2>/dev/null)
      if [[ -n "$pr_json" ]]; then
        echo "$pr_json" > "$pr_cache"
      else
        # No PR for this branch - cache empty result
        echo '{"state":"NONE"}' > "$pr_cache"
      fi
    fi
  fi

  # Parse cached PR status
  if [[ -f "$pr_cache" ]]; then
    eval "$(jq -r '
      @sh "pr_state=\(.state // "NONE")",
      @sh "pr_review=\(.reviewDecision // "")",
      @sh "pr_draft=\(.isDraft // false)"
    ' "$pr_cache" 2>/dev/null)" || {
      pr_state="NONE"
      pr_review=""
      pr_draft=false
    }

    # Build PR status display (colored dot + label)
    if [[ "$pr_state" != "NONE" ]]; then
      if [[ "$pr_draft" == "true" ]]; then
        pr_display="${C_PR_DRAFT}◌ draft${C_RESET}"
      elif [[ "$pr_review" == "APPROVED" ]]; then
        pr_display="${C_PR_APPROVED}● approved${C_RESET}"
      elif [[ "$pr_review" == "CHANGES_REQUESTED" ]]; then
        pr_display="${C_PR_CHANGES}● changes${C_RESET}"
      elif [[ "$pr_state" == "OPEN" ]]; then
        pr_display="${C_PR_PENDING}● pending${C_RESET}"
      elif [[ "$pr_state" == "MERGED" ]]; then
        pr_display="${C_PR_APPROVED}● merged${C_RESET}"
      fi
    fi
  fi
fi

# --- Context Calculation ---
ctx_no_data=false
if [[ $current_usage -gt 0 && $context_size -gt 0 ]]; then
  ctx_pct=$((current_usage * 100 / context_size))
elif [[ -n "$used_pct" ]]; then
  ctx_pct=${used_pct%.*}
  current_usage=$((ctx_pct * context_size / 100))
else
  # No data available (fresh session start)
  ctx_pct=0
  ctx_no_data=true
fi

# Clamp context percentage to 0-100 range
[[ $ctx_pct -lt 0 ]] && ctx_pct=0
[[ $ctx_pct -gt 100 ]] && ctx_pct=100

# Progress bar (pure bash, no subshells)
bar_width=8
filled=$((ctx_pct * bar_width / 100))
[[ $filled -gt $bar_width ]] && filled=$bar_width
[[ $filled -lt 0 ]] && filled=0
empty=$((bar_width - filled))

bar=$(build_bar "$filled" "$empty")

ctx_color=$(get_semantic_color "$ctx_pct")
if [[ "$ctx_no_data" == "true" ]]; then
  ctx_tokens="—"
else
  ctx_tokens=$(format_tokens "$current_usage")
fi
ctx_max=$(format_tokens "$context_size")

# --- Usage Stats (cached 60s, errors cached 15s) ---
usage_5h=""
usage_7d=""
usage_extra=""

usage_cache_age=999
if [[ -f "$usage_cache" ]]; then
  usage_cache_age=$(( NOW - $(stat_mtime "$usage_cache") ))
fi

# Check if cache contains an error (shorter TTL for errors)
usage_is_error=false
if [[ -f "$usage_cache" ]]; then
  # Quick check: valid response has "five_hour" key
  if ! grep -q '"five_hour"' "$usage_cache" 2>/dev/null; then
    usage_is_error=true
  fi
fi

# Refresh if: cache expired (60s) OR error cache expired (15s)
if [[ $usage_cache_age -gt 60 ]] || { [[ "$usage_is_error" == "true" ]] && [[ $usage_cache_age -gt 15 ]]; }; then
  # Get OAuth token: macOS uses keychain, Linux uses credentials file
  if [[ "$IS_MACOS" == "true" ]]; then
    token=$(security find-generic-password -s "Claude Code-credentials" -w 2>/dev/null | jq -r '.claudeAiOauth.accessToken' 2>/dev/null)
  else
    token=$(jq -r '.claudeAiOauth.accessToken' ~/.claude/.credentials.json 2>/dev/null)
  fi
  if [[ -n "$token" && "$token" != "null" ]]; then
    usage_json=$(curl -s -m 2 -H "Authorization: Bearer $token" -H "anthropic-beta: oauth-2025-04-20" "https://api.anthropic.com/api/oauth/usage" 2>/dev/null)
    # Only cache if we got a response (even errors, for rate limiting)
    [[ -n "$usage_json" ]] && echo "$usage_json" > "$usage_cache"
  fi
fi

if [[ -f "$usage_cache" ]] && grep -q '"five_hour"' "$usage_cache" 2>/dev/null; then
  # Single jq call for all usage data
  eval "$(jq -r '
    @sh "five_util=\(.five_hour.utilization // 0)",
    @sh "five_reset=\(.five_hour.resets_at // "")",
    @sh "seven_util=\(.seven_day.utilization // 0)",
    @sh "seven_reset=\(.seven_day.resets_at // "")",
    @sh "extra_enabled=\(.extra_usage.is_enabled // false)",
    @sh "extra_util=\(.extra_usage.utilization // 0)",
    @sh "extra_used=\(.extra_usage.used_credits // 0)",
    @sh "extra_limit=\(.extra_usage.monthly_limit // 0)"
  ' "$usage_cache" 2>/dev/null)" || {
    five_util=0
    seven_util=0
    five_reset=""
    seven_reset=""
    extra_enabled=false
    extra_util=0
    extra_used=0
    extra_limit=0
  }

  # Validate and convert to integers
  five_util_int=${five_util%.*}
  seven_util_int=${seven_util%.*}
  [[ ! "$five_util_int" =~ ^[0-9]+$ ]] && five_util_int=0
  [[ ! "$seven_util_int" =~ ^[0-9]+$ ]] && seven_util_int=0

  # Clamp to 0-100
  [[ $five_util_int -gt 100 ]] && five_util_int=100
  [[ $seven_util_int -gt 100 ]] && seven_util_int=100

  five_color=$(get_semantic_color "$five_util_int")
  seven_color=$(get_semantic_color "$seven_util_int")

  five_time=$(format_reset_time "$five_reset")
  seven_time=$(format_reset_time "$seven_reset")

  usage_5h="${five_color}5h:${five_util_int}% ${C_MUTED}(${five_time})${C_RESET}"
  usage_7d="${seven_color}7d:${seven_util_int}% ${C_MUTED}(${seven_time})${C_RESET}"

  # Extra usage (only show when enabled)
  if [[ "$extra_enabled" == "true" ]]; then
    extra_util_int=${extra_util%.*}
    extra_used_int=${extra_used%.*}
    extra_color=$(get_semantic_color "$extra_util_int")
    usage_extra="${extra_color}Extra:${extra_util_int}% ${C_MUTED}(\$${extra_used_int}/\$${extra_limit})${C_RESET}"
  fi
fi

# --- Session Duration (from Claude Code's cost.total_duration_ms) ---
duration_secs=$((duration_ms / 1000))
duration_display=$(format_duration "$duration_secs")

# --- Lines Changed ---
lines_display=""
if [[ $lines_added -gt 0 || $lines_removed -gt 0 ]]; then
  lines_display="${C_GIT_ADD}+${lines_added}${C_RESET}/${C_GIT_DEL}-${lines_removed}${C_RESET}"
fi

# --- Session Cost ---
cost_display=""
if [[ -n "$total_cost" && "$total_cost" != "0" ]]; then
  # Format cost based on magnitude
  cost_int=${total_cost%.*}
  [[ -z "$cost_int" ]] && cost_int=0
  if [[ $cost_int -ge 10 ]]; then
    # $10+ -> show as integer: $12
    cost_display="${C_MUTED}\$${cost_int}${C_RESET}"
  else
    # < $10 -> show with decimals: $0.23 or $1.50
    cost_rounded=$(printf "%.2f" "$total_cost")
    cost_display="${C_MUTED}\$${cost_rounded}${C_RESET}"
  fi
fi

# --- Build Output ---
sep=" ${C_MUTED}│${C_RESET} "

row1="${C_ACCENT}${model_short}${C_RESET}"
[[ -n "$dir_display" ]] && row1+="${sep}${C_WHITE}${dir_display}${C_RESET}"
[[ -n "$git_branch" ]] && row1+="${sep}${C_ACCENT}${git_branch}${C_RESET}${git_status}"
[[ -n "$pr_display" ]] && row1+="${sep}${pr_display}"
[[ -n "$lines_display" ]] && row1+="${sep}${lines_display}"

row2="${ctx_color}${bar} ${ctx_tokens}/${ctx_max}${C_RESET}"
[[ -n "$usage_5h" ]] && row2+="${sep}${usage_5h}"
[[ -n "$usage_7d" ]] && row2+="${sep}${usage_7d}"
[[ -n "$usage_extra" ]] && row2+="${sep}${usage_extra}"
[[ -n "$cost_display" ]] && row2+="${sep}${cost_display}"
row2+="${sep}${C_MUTED}${duration_display}${C_RESET}"

printf "%b\n%b" "$row1" "$row2"
