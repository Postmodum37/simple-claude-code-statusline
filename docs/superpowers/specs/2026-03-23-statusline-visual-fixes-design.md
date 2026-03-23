# Statusline Visual Fixes

Two focused fixes to improve readability of the second statusline row.

## 1. Progress Bar Characters

**Problem:** Current `▰`/`▱` (small parallelogram) characters are hard to read in many terminal fonts — low contrast and small glyph size.

**Fix:** Replace with `▓` (medium shade) for filled and `░` (light shade) for empty. Higher contrast, better readability, universally supported.

**Before:** `▰▰▰▰▰▰▰▰▰▱▱▱▱▱▱▱▱▱▱▱`
**After:** `▓▓▓▓▓▓▓▓▓░░░░░░░░░░`

**Scope:** `render.go` — `buildProgressBar` function, lines 79 and 87. Swap two character literals. Auto-compact marker (`│`) unchanged.

## 2. Extra Usage Dollar Display

**Problem:** The usage API returns `used_credits` and `monthly_limit` in cents (e.g., `3258.0` and `30000`). The code displays them raw with a `$` prefix, producing inflated values like `$3258/$30000` instead of `$32/$300`.

**Fix:** Divide both values by 100 before display. Show as rounded integers (consistent with the existing `FormatCost` behavior above $10).

**Before:** `Extra:11% ($3258/$30000)`
**After:** `Extra:11% ($32/$300)`

**Scope:** `render.go` — `buildUsageSection` function, line 335-338. Change the format expression to divide by 100.

## Files Changed

- `src/render.go` — two changes (progress bar chars, extra usage division)
- `src/render_test.go` — update test input data to use cent-denominated values (e.g., `UsedCredits: 5000.0`, `MonthlyLimit: 20000.0`) and update expected output strings

## Testing

Run `go test ./src/...` after changes. Manually verify with:
```sh
echo '{"model":{"id":"claude-opus-4-6"},"cwd":"/tmp","context_window":{"used_percentage":45,"context_window_size":200000}}' | ./bin/statusline.sh
```
