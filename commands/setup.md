---
name: setup
description: Install and configure the simple-statusline for Claude Code
---

# Simple Statusline Setup

You are helping the user install the simple-statusline plugin.

## Steps to perform:

1. **Find the plugin's statusline.sh script**
   - Look for `bin/statusline.sh` in the plugin directory (the directory containing `.claude-plugin/plugin.json`)
   - The plugin is installed somewhere under `~/.claude/plugins/`

2. **Copy the script to ~/.claude/**
   ```bash
   cp <plugin-path>/bin/statusline.sh ~/.claude/statusline.sh
   chmod +x ~/.claude/statusline.sh
   ```

3. **Update settings.json**
   - Read `~/.claude/settings.json`
   - Add or update the `statusLine` configuration:
   ```json
   {
     "statusLine": {
       "type": "command",
       "command": "~/.claude/statusline.sh"
     }
   }
   ```
   - Preserve all other existing settings

4. **Confirm success**
   - Tell the user the statusline is now configured
   - Let them know they may need to restart Claude Code to see changes
   - Mention they can edit `~/.claude/statusline.sh` directly to customize

## Requirements

The script requires:
- `jq` for JSON parsing
- `curl` for API rate limit fetching
- `git` for repository status (optional)

If jq is not installed, suggest: `brew install jq`
