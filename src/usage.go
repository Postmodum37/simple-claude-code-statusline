package main

import (
	"time"
)

// UsageData holds rate limit data for display.
type UsageData struct {
	FiveHour *UsageWindow `json:"five_hour,omitempty"`
	SevenDay *UsageWindow `json:"seven_day,omitempty"`
}

// UsageWindow represents a single rate limit window (5h or 7d).
type UsageWindow struct {
	Utilization float64 `json:"utilization"`
	ResetsAt    string  `json:"resets_at"`
}

// GetUsageData converts stdin rate limits to UsageData.
// Returns nil if no rate limits are available.
func GetUsageData(stdin *StdinData) *UsageData {
	if stdin.RateLimits == nil {
		return nil
	}

	data := &UsageData{}

	if stdin.RateLimits.FiveHour != nil {
		uw := &UsageWindow{}
		if stdin.RateLimits.FiveHour.UsedPercentage != nil {
			uw.Utilization = *stdin.RateLimits.FiveHour.UsedPercentage
		}
		if stdin.RateLimits.FiveHour.ResetsAt != nil {
			uw.ResetsAt = time.Unix(int64(*stdin.RateLimits.FiveHour.ResetsAt), 0).UTC().Format(time.RFC3339)
		}
		data.FiveHour = uw
	}

	if stdin.RateLimits.SevenDay != nil {
		uw := &UsageWindow{}
		if stdin.RateLimits.SevenDay.UsedPercentage != nil {
			uw.Utilization = *stdin.RateLimits.SevenDay.UsedPercentage
		}
		if stdin.RateLimits.SevenDay.ResetsAt != nil {
			uw.ResetsAt = time.Unix(int64(*stdin.RateLimits.SevenDay.ResetsAt), 0).UTC().Format(time.RFC3339)
		}
		data.SevenDay = uw
	}

	return data
}
