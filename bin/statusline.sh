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

# --- Read JSON input ---
input=$(cat)

# --- Cache files ---
context_cache="/tmp/claude-context-cache"
git_cache="/tmp/claude-git-cache"
usage_cache="/tmp/claude-usage-cache"

# --- Extract all fields with single jq call ---
eval "$(echo "$input" | jq -r '
  @sh "model_id=\(.model.id // "")",
  @sh "model_display=\(.model.display_name // "")",
  @sh "project_dir=\(.workspace.project_dir // "")",
  @sh "cwd=\(.cwd // "")",
  @sh "context_size=\(.context_window.context_window_size // 200000)",
  @sh "used_pct=\(.context_window.used_percentage // "")",
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
    [[ $mins -gt 0 ]] && echo "${hours}h${mins}m" || echo "${hours}h"
  else
    local days=$((diff / 86400))
    local hours=$(((diff % 86400) / 3600))
    [[ $hours -gt 0 ]] && echo "${days}d${hours}h" || echo "${days}d"
  fi
}

# --- Model Display (no subshells) ---
model_short=""
case "$model_id" in
  *opus*)   model_short="Opus" ;;
  *sonnet*) model_short="Sonnet" ;;
  *haiku*)  model_short="Haiku" ;;
  *)        model_short="${model_display%% *}" ;;
esac

# Extract version using bash parameter expansion (no sed)
if [[ "$model_id" =~ -([0-9])-([0-9])- ]]; then
  model_short="$model_short ${BASH_REMATCH[1]}.${BASH_REMATCH[2]}"
fi

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
    # Get last 3 segments
    IFS='/' read -ra parts <<< "$dir_display"
    len=${#parts[@]}
    dir_display="…/${parts[$((len-3))]}/${parts[$((len-2))]}/${parts[$((len-1))]}"
  fi
fi

# --- Git Display (cached for 5 seconds) ---
git_branch=""
git_status=""

if [[ -n "$project_dir" ]]; then
  git_cache_age=999
  if [[ -f "$git_cache" ]]; then
    git_cache_age=$(( $(date +%s) - $(stat -f %m "$git_cache" 2>/dev/null || echo 0) ))
  fi

  if [[ $git_cache_age -gt 5 ]]; then
    # Refresh git cache
    if git --no-optional-locks -C "$project_dir" rev-parse --git-dir &>/dev/null; then
      git_branch=$(git --no-optional-locks -C "$project_dir" rev-parse --abbrev-ref HEAD 2>/dev/null)
      [[ ${#git_branch} -gt 20 ]] && git_branch="${git_branch:0:17}…"

      git_porcelain=$(git --no-optional-locks -C "$project_dir" status --porcelain 2>/dev/null)
      added=0; modified=0; deleted=0
      while IFS= read -r line; do
        case "${line:0:2}" in
          "??"|"A ") ((added++)) ;;
          " M"|"M "|"MM") ((modified++)) ;;
          " D"|"D ") ((deleted++)) ;;
        esac
      done <<< "$git_porcelain"

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

# Progress bar (no loop)
bar_width=8
filled=$((ctx_pct * bar_width / 100))
[[ $filled -gt $bar_width ]] && filled=$bar_width
empty=$((bar_width - filled))

bar=$(printf '%*s' "$filled" '' | tr ' ' '▰')$(printf '%*s' "$empty" '' | tr ' ' '▱')

ctx_color=$(get_semantic_color "$ctx_pct")
if [[ "$ctx_no_data" == "true" ]]; then
  ctx_tokens="—"
else
  ctx_tokens=$(format_tokens "$current_usage")
fi
ctx_max=$(format_tokens "$context_size")

# --- Usage Stats (cached 60s) ---
usage_5h=""
usage_7d=""
usage_extra=""

usage_cache_age=999
if [[ -f "$usage_cache" ]]; then
  usage_cache_age=$(( $(date +%s) - $(stat -f %m "$usage_cache" 2>/dev/null || echo 0) ))
fi

if [[ $usage_cache_age -gt 60 ]]; then
  token=$(security find-generic-password -s "Claude Code-credentials" -w 2>/dev/null | jq -r '.claudeAiOauth.accessToken' 2>/dev/null)
  if [[ -n "$token" && "$token" != "null" ]]; then
    usage_json=$(curl -s -m 2 -H "Authorization: Bearer $token" -H "anthropic-beta: oauth-2025-04-20" "https://api.anthropic.com/api/oauth/usage" 2>/dev/null)
    [[ -n "$usage_json" ]] && echo "$usage_json" > "$usage_cache"
  fi
fi

if [[ -f "$usage_cache" ]]; then
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
  ' "$usage_cache" 2>/dev/null)"

  five_util_int=${five_util%.*}
  seven_util_int=${seven_util%.*}

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

# --- Session Duration ---
session_file="/tmp/claude-session-${session_id:-$$}"
[[ ! -f "$session_file" ]] && date +%s > "$session_file"
start_time=$(cat "$session_file" 2>/dev/null || date +%s)
duration_secs=$(( $(date +%s) - start_time ))
duration_display=$(format_duration "$duration_secs")

# --- Build Output ---
sep=" ${C_MUTED}│${C_RESET} "

row1="${C_ACCENT}${model_short}${C_RESET}"
[[ -n "$dir_display" ]] && row1+="${sep}${C_WHITE}${dir_display}${C_RESET}"
[[ -n "$git_branch" ]] && row1+="${sep}${C_ACCENT}${git_branch}${C_RESET}${git_status}"

row2="${ctx_color}${bar} ${ctx_tokens}/${ctx_max}${C_RESET}"
[[ -n "$usage_5h" ]] && row2+="${sep}${usage_5h}"
[[ -n "$usage_7d" ]] && row2+="${sep}${usage_7d}"
[[ -n "$usage_extra" ]] && row2+="${sep}${usage_extra}"
row2+="${sep}${C_MUTED}${duration_display}${C_RESET}"

printf "%b\n%b" "$row1" "$row2"
