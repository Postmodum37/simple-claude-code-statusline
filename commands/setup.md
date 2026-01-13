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

2. **Create symlink at ~/.claude/statusline.sh**
   - Find the plugin's script at `${CLAUDE_PLUGIN_ROOT}/bin/statusline.sh`
   - Create symlink: `ln -sf "${CLAUDE_PLUGIN_ROOT}/bin/statusline.sh" ~/.claude/statusline.sh`
   - This allows plugin updates to work automatically

3. **Read ~/.claude/settings.json**
   - Create the file with `{}` if it doesn't exist

4. **Add/update the statusLine config:**
   ```json
   {
     "statusLine": {
       "type": "command",
       "command": "~/.claude/statusline.sh"
     }
   }
   ```
   - Preserve all other existing settings
   - The symlink points to the plugin, so updates work automatically

5. **Confirm success**
   - Tell the user the statusline is configured
   - Mention that future plugin updates will work automatically
