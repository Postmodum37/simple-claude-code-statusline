package main

import (
	"testing"
	"time"
)

func TestGetUsageDataBothWindows(t *testing.T) {
	fiveHourPct := 42.0
	fiveHourResets := float64(time.Date(2026, 3, 23, 18, 0, 0, 0, time.UTC).Unix())
	sevenDayPct := 15.0
	sevenDayResets := float64(time.Date(2026, 3, 28, 0, 0, 0, 0, time.UTC).Unix())

	stdin := &StdinData{
		RateLimits: &RateLimits{
			FiveHour: &RateLimitWindow{
				UsedPercentage: &fiveHourPct,
				ResetsAt:       &fiveHourResets,
			},
			SevenDay: &RateLimitWindow{
				UsedPercentage: &sevenDayPct,
				ResetsAt:       &sevenDayResets,
			},
		},
	}

	data := GetUsageData(stdin)
	if data == nil {
		t.Fatal("GetUsageData returned nil")
	}
	if data.FiveHour == nil {
		t.Fatal("FiveHour is nil")
	}
	if data.FiveHour.Utilization != 42.0 {
		t.Errorf("FiveHour.Utilization = %f, want 42.0", data.FiveHour.Utilization)
	}
	if data.FiveHour.ResetsAt != "2026-03-23T18:00:00Z" {
		t.Errorf("FiveHour.ResetsAt = %q, want %q", data.FiveHour.ResetsAt, "2026-03-23T18:00:00Z")
	}
	if data.SevenDay == nil {
		t.Fatal("SevenDay is nil")
	}
	if data.SevenDay.Utilization != 15.0 {
		t.Errorf("SevenDay.Utilization = %f, want 15.0", data.SevenDay.Utilization)
	}
	if data.SevenDay.ResetsAt != "2026-03-28T00:00:00Z" {
		t.Errorf("SevenDay.ResetsAt = %q, want %q", data.SevenDay.ResetsAt, "2026-03-28T00:00:00Z")
	}
}

func TestGetUsageDataOneWindow(t *testing.T) {
	fiveHourPct := 42.5
	fiveHourResets := float64(time.Date(2026, 3, 23, 18, 0, 0, 0, time.UTC).Unix())

	stdin := &StdinData{
		RateLimits: &RateLimits{
			FiveHour: &RateLimitWindow{
				UsedPercentage: &fiveHourPct,
				ResetsAt:       &fiveHourResets,
			},
		},
	}

	data := GetUsageData(stdin)
	if data == nil {
		t.Fatal("GetUsageData returned nil")
	}
	if data.FiveHour == nil {
		t.Fatal("FiveHour is nil")
	}
	if data.FiveHour.Utilization != 42.5 {
		t.Errorf("FiveHour.Utilization = %f, want 42.5", data.FiveHour.Utilization)
	}
	if data.SevenDay != nil {
		t.Error("SevenDay should be nil when not provided")
	}
}

func TestGetUsageDataNoRateLimits(t *testing.T) {
	stdin := &StdinData{}
	data := GetUsageData(stdin)
	if data != nil {
		t.Errorf("GetUsageData with no rate limits should return nil, got %+v", data)
	}
}

func TestGetUsageDataNilUsedPercentage(t *testing.T) {
	resets := float64(time.Date(2026, 3, 23, 18, 0, 0, 0, time.UTC).Unix())

	stdin := &StdinData{
		RateLimits: &RateLimits{
			FiveHour: &RateLimitWindow{
				UsedPercentage: nil,
				ResetsAt:       &resets,
			},
		},
	}

	data := GetUsageData(stdin)
	if data == nil {
		t.Fatal("GetUsageData returned nil")
	}
	if data.FiveHour == nil {
		t.Fatal("FiveHour is nil")
	}
	if data.FiveHour.Utilization != 0 {
		t.Errorf("FiveHour.Utilization = %f, want 0 (nil percentage)", data.FiveHour.Utilization)
	}
	if data.FiveHour.ResetsAt != "2026-03-23T18:00:00Z" {
		t.Errorf("FiveHour.ResetsAt = %q, want %q", data.FiveHour.ResetsAt, "2026-03-23T18:00:00Z")
	}
}
