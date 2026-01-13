---
name: setup
description: Configure the statusline in Claude Code settings
allowed-tools:
  - Bash
  - Read
  - Edit
  - Write
---

# Simple Statusline Setup

Configure the user's statusline setting to use this plugin.

## Steps:

1. **Check jq is installed**
   - Run `which jq` to verify
   - If not installed, tell the user: `brew install jq`

2. **Read ~/.claude/settings.json**
   - Create the file with `{}` if it doesn't exist

3. **Add/update the statusLine config:**
   ```json
   {
     "statusLine": {
       "type": "command",
       "command": "${CLAUDE_PLUGIN_ROOT}/bin/statusline.sh"
     }
   }
   ```
   - Preserve all other existing settings
   - `${CLAUDE_PLUGIN_ROOT}` is resolved by Claude Code to the plugin's cache directory

4. **Confirm success**
   - Tell the user the statusline is configured
   - Remind them to restart Claude Code to see changes
