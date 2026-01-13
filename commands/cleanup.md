---
name: cleanup
description: Remove statusline configuration before uninstalling the plugin
allowed-tools:
  - Bash
  - Read
  - Edit
---

# Simple Statusline Cleanup

Remove the statusline configuration from settings so the user can uninstall the plugin cleanly.

## Steps:

1. **Read ~/.claude/settings.json**

2. **Remove the statusLine config:**
   - Find and remove the entire `"statusLine": { ... }` object from the JSON
   - Preserve all other settings

3. **Remove the symlink:**
   - Run: `rm -f ~/.claude/statusline.sh`

4. **Confirm success:**
   - Tell the user the statusline config and symlink have been removed
   - They can now safely run: `/plugin uninstall simple-statusline`
