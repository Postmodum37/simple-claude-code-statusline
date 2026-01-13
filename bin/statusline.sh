#!/bin/bash

# Claude Code Statusline V2 - Tokyo Night Style
# Features: Usage stats, full git status, adaptive collapse

# --- Tokyo Night Color Palette ---
C_RESET="\033[0m"
C_BG1="\033[48;5;238m"        # #394260 - model bg
C_BG2="\033[48;5;235m"        # #212736 - dir bg
C_BG3="\033[48;5;234m"        # #1d2230 - git bg
C_FG1="\033[38;5;238m"        # #394260 - for separators
C_FG2="\033[38;5;235m"        # #212736
C_FG3="\033[38;5;234m"        # #1d2230
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

# Separator style
SEP_PIPE="│"

# --- Read JSON input ---
input=$(cat)

# --- Context cache (prevents flicker during UI operations) ---
context_cache="/tmp/claude-context-cache"

# --- Extract all fields with single jq call ---
eval "$(echo "$input" | jq -r '
  @sh "model_id=\(.model.id // "")",
  @sh "model_display=\(.model.display_name // "")",
  @sh "project_dir=\(.workspace.project_dir // "")",
  @sh "cwd=\(.cwd // "")",
  @sh "context_size=\(.context_window.context_window_size // 200000)",
  @sh "total_input=\(.context_window.total_input_tokens // 0)",
  @sh "total_output=\(.context_window.total_output_tokens // 0)",
  @sh "used_pct=\(.context_window.used_percentage // 0)",
  @sh "session_id=\(.session_id // "")",
  @sh "current_usage=\(
    (.context_window.current_usage.input_tokens // 0) +
    (.context_window.current_usage.output_tokens // 0) +
    (.context_window.current_usage.cache_creation_input_tokens // 0) +
    (.context_window.current_usage.cache_read_input_tokens // 0)
  )"
' 2>/dev/null)" || {
  model_id=""
  current_usage=0
  context_size=200000
}

# --- Use cached context if current parse returned zero ---
if [[ $current_usage -eq 0 && -f "$context_cache" ]]; then
  source "$context_cache" 2>/dev/null
elif [[ $current_usage -gt 0 ]]; then
  # Cache valid context values
  echo "current_usage=$current_usage; context_size=$context_size; used_pct=$used_pct" > "$context_cache"
fi

# --- Terminal width ---
term_width="${COLUMNS:-$(tput cols 2>/dev/null || echo 100)}"

# --- Helper Functions ---

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

get_semantic_color() {
  local pct=$1
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

format_duration() {
  local secs=$1
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
  local now_utc=$(date -u +%s)

  # Strip fractional seconds and timezone suffix, parse as UTC
  local reset_clean="${reset_iso%%.*}"
  local reset_ts=$(date -j -u -f "%Y-%m-%dT%H:%M:%S" "$reset_clean" +%s 2>/dev/null || echo "$now_utc")
  local diff=$((reset_ts - now_utc))

  if [[ $diff -lt 0 ]]; then
    echo "0m"
  elif [[ $diff -lt 3600 ]]; then
    echo "$((diff / 60))m"
  elif [[ $diff -lt 86400 ]]; then
    local hours=$((diff / 3600))
    local mins=$(((diff % 3600) / 60))
    if [[ $mins -gt 0 ]]; then
      echo "${hours}h${mins}m"
    else
      echo "${hours}h"
    fi
  else
    echo "$((diff / 86400))d"
  fi
}

# --- Model Display ---
model_short=""
if [[ "$model_id" == *"opus"* ]]; then
  model_short="Opus"
elif [[ "$model_id" == *"sonnet"* ]]; then
  model_short="Sonnet"
elif [[ "$model_id" == *"haiku"* ]]; then
  model_short="Haiku"
elif [[ -n "$model_display" ]]; then
  model_short=$(echo "$model_display" | awk '{print $1}')
fi

# Extract version (e.g., "4.5" from "claude-opus-4-5-20251101")
# Use sed since BASH_REMATCH is buggy in bash 3.2 (macOS default)
version=$(echo "$model_id" | sed -n 's/.*-\([0-9]\)-\([0-9]\)-.*/\1.\2/p')
if [[ -n "$version" ]]; then
  model_short="$model_short $version"
fi

# --- Directory Display (~ substitution + truncate to 3 segments) ---
dir_display=""
if [[ -n "$cwd" ]]; then
  # Replace home directory with ~
  dir_display="${cwd/#$HOME/~}"

  # Count path segments and truncate if needed
  # Use a simpler approach compatible with bash 3.x
  segment_count=$(echo "$dir_display" | tr '/' '\n' | grep -c .)

  if [[ $segment_count -gt 3 ]]; then
    # Get last 3 segments
    last_three=$(echo "$dir_display" | rev | cut -d'/' -f1-3 | rev)
    dir_display="…/${last_three}"
  fi
fi

# --- Git Display (full status) ---
# Use --no-optional-locks to avoid conflicts with other git processes
git_branch=""
git_status=""
if [[ -n "$project_dir" ]] && git --no-optional-locks -C "$project_dir" rev-parse --git-dir &>/dev/null 2>&1; then
  # Branch name (up to 20 chars)
  git_branch=$(git --no-optional-locks -C "$project_dir" rev-parse --abbrev-ref HEAD 2>/dev/null)
  if [[ ${#git_branch} -gt 20 ]]; then
    git_branch="${git_branch:0:17}…"
  fi

  # File changes (single status call, parsed for all counts)
  git_porcelain=$(git --no-optional-locks -C "$project_dir" status --porcelain 2>/dev/null)
  if [[ -n "$git_porcelain" ]]; then
    added=$(echo "$git_porcelain" | grep -c '^??\|^A ' 2>/dev/null); added=${added:-0}
    modified=$(echo "$git_porcelain" | grep -c '^ M\|^M \|^MM' 2>/dev/null); modified=${modified:-0}
    deleted=$(echo "$git_porcelain" | grep -c '^ D\|^D ' 2>/dev/null); deleted=${deleted:-0}
  else
    added=0
    modified=0
    deleted=0
  fi

  # Ahead/behind
  ahead=$(git --no-optional-locks -C "$project_dir" rev-list --count @{u}..HEAD 2>/dev/null || echo 0)
  behind=$(git --no-optional-locks -C "$project_dir" rev-list --count HEAD..@{u} 2>/dev/null || echo 0)

  # Build status string
  git_status=""
  [[ $added -gt 0 ]] && git_status+=" ${C_GIT_ADD}✚${added}${C_RESET}"
  [[ $modified -gt 0 ]] && git_status+=" ${C_GIT_MOD}●${modified}${C_RESET}"
  [[ $deleted -gt 0 ]] && git_status+=" ${C_GIT_DEL}✖${deleted}${C_RESET}"
  [[ $ahead -gt 0 ]] && git_status+=" ${C_GIT_AHEAD}↑${ahead}${C_RESET}"
  [[ $behind -gt 0 ]] && git_status+=" ${C_GIT_BEHIND}↓${behind}${C_RESET}"
fi

# --- Context Calculation ---
if [[ $current_usage -gt 0 && $context_size -gt 0 ]]; then
  ctx_pct=$((current_usage * 100 / context_size))
elif [[ -n "$used_pct" && "$used_pct" != "0" ]]; then
  ctx_pct=${used_pct%.*}
  current_usage=$((ctx_pct * context_size / 100))
else
  ctx_pct=0
fi

# Progress bar
bar_width=8
filled=$((ctx_pct * bar_width / 100))
[[ $filled -gt $bar_width ]] && filled=$bar_width
empty=$((bar_width - filled))

bar=""
for ((i=0; i<filled; i++)); do bar+="▰"; done
for ((i=0; i<empty; i++)); do bar+="▱"; done

ctx_color=$(get_semantic_color "$ctx_pct")
ctx_tokens=$(format_tokens "$current_usage")
ctx_max=$(format_tokens "$context_size")

# --- Usage Stats (with caching) ---
cache_file="/tmp/claude-usage-cache"
cache_age=999

if [[ -f "$cache_file" ]]; then
  cache_age=$(( $(date +%s) - $(stat -f %m "$cache_file" 2>/dev/null || echo 0) ))
fi

usage_5h=""
usage_7d=""

if [[ $cache_age -gt 60 ]]; then
  # Fetch fresh usage data
  token=$(security find-generic-password -s "Claude Code-credentials" -w 2>/dev/null | jq -r '.claudeAiOauth.accessToken' 2>/dev/null)
  if [[ -n "$token" && "$token" != "null" ]]; then
    usage_json=$(curl -s -m 2 -H "Authorization: Bearer $token" -H "anthropic-beta: oauth-2025-04-20" "https://api.anthropic.com/api/oauth/usage" 2>/dev/null)
    if [[ -n "$usage_json" ]]; then
      echo "$usage_json" > "$cache_file"
    fi
  fi
fi

if [[ -f "$cache_file" ]]; then
  five_util=$(jq -r '.five_hour.utilization // 0' "$cache_file" 2>/dev/null)
  five_reset=$(jq -r '.five_hour.resets_at // ""' "$cache_file" 2>/dev/null)
  seven_util=$(jq -r '.seven_day.utilization // 0' "$cache_file" 2>/dev/null)
  seven_reset=$(jq -r '.seven_day.resets_at // ""' "$cache_file" 2>/dev/null)

  five_util_int=${five_util%.*}
  seven_util_int=${seven_util%.*}

  five_color=$(get_semantic_color "$five_util_int")
  seven_color=$(get_semantic_color "$seven_util_int")

  five_time=$(format_reset_time "$five_reset")
  seven_time=$(format_reset_time "$seven_reset")

  usage_5h="${five_color}5h:${five_util_int}%${C_MUTED}(${five_time})${C_RESET}"
  usage_7d="${seven_color}7d:${seven_util_int}%${C_MUTED}(${seven_time})${C_RESET}"
fi

# --- Session Duration ---
session_file="/tmp/claude-session-${session_id:-$$}"
if [[ ! -f "$session_file" ]]; then
  date +%s > "$session_file"
fi
start_time=$(cat "$session_file" 2>/dev/null || date +%s)
duration_secs=$(( $(date +%s) - start_time ))
duration_display=$(format_duration "$duration_secs")

# --- Build Segments ---

sep=" ${C_MUTED}${SEP_PIPE}${C_RESET} "

seg_model="${C_ACCENT}${model_short}${C_RESET}"
seg_dir="${C_WHITE}${dir_display}${C_RESET}"
seg_git="${C_ACCENT}${git_branch}${C_RESET}${git_status}"
seg_context="${ctx_color}${bar} ${ctx_tokens}/${ctx_max}${C_RESET}"
seg_usage_5h="$usage_5h"
seg_usage_7d="$usage_7d"
seg_duration="${C_MUTED}${duration_display}${C_RESET}"

# --- Two-Row Layout ---
# Row 1: Model | Directory | Git
# Row 2: Context | 5h usage | 7d usage | Duration

row1="$seg_model"
[[ -n "$dir_display" ]] && row1+="${sep}${seg_dir}"
[[ -n "$git_branch" ]] && row1+="${sep}${seg_git}"

row2="$seg_context"
[[ -n "$usage_5h" ]] && row2+="${sep}${seg_usage_5h}"
[[ -n "$usage_7d" ]] && row2+="${sep}${seg_usage_7d}"
row2+="${sep}${seg_duration}"

printf "%b\n%b" "$row1" "$row2"
