#!/bin/bash

SETTINGS="$HOME/.claude/settings.json"

# Only notify if statusLine not configured
if [ ! -f "$SETTINGS" ] || ! grep -q '"statusLine"' "$SETTINGS"; then
  echo "Run /simple-statusline:setup to enable the statusline"
fi

exit 0
