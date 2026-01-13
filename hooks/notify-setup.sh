#!/bin/bash

SETTINGS="$HOME/.claude/settings.json"
SYMLINK="$HOME/.claude/statusline.sh"

# Auto-update symlink if it exists (keeps it pointing to current version after plugin updates)
if [ -L "$SYMLINK" ]; then
  ln -sf "${CLAUDE_PLUGIN_ROOT}/bin/statusline.sh" "$SYMLINK"
fi

# Only notify if statusLine not configured
if [ ! -f "$SETTINGS" ] || ! grep -q '"statusLine"' "$SETTINGS"; then
  echo "Run /simple-statusline:setup to enable the statusline"
fi

exit 0
