package main

import (
	"testing"
	"time"
)

func TestFormatTokens(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{0, "0"},
		{500, "500"},
		{1000, "1k"},
		{42000, "42k"},
		{200000, "200k"},
		{1000000, "1m"},
		{1048576, "1m"},
	}
	for _, tt := range tests {
		got := FormatTokens(tt.input)
		if got != tt.want {
			t.Errorf("FormatTokens(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{0, "0m"},
		{90, "1m"},
		{3600, "1h0m"},
		{3660, "1h1m"},
		{86400, "1d0h"},
		{90061, "1d1h"},
		{-5, "0m"},
	}
	for _, tt := range tests {
		got := FormatDuration(tt.input)
		if got != tt.want {
			t.Errorf("FormatDuration(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFormatCost(t *testing.T) {
	tests := []struct {
		input float64
		want  string
	}{
		{0.0, ""},
		{0.23, "$0.23"},
		{1.5, "$1.50"},
		{9.99, "$9.99"},
		{10.0, "$10"},
		{12.7, "$12"},
	}
	for _, tt := range tests {
		got := FormatCost(tt.input)
		if got != tt.want {
			t.Errorf("FormatCost(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFormatResetTime(t *testing.T) {
	now := time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		resetISO string
		want     string
	}{
		{"30 min from now", now.Add(30 * time.Minute).Format(time.RFC3339), "30m"},
		{"2h15m from now", now.Add(2*time.Hour + 15*time.Minute).Format(time.RFC3339), "2h15m"},
		{"2h exactly", now.Add(2 * time.Hour).Format(time.RFC3339), "2h"},
		{"1d3h from now", now.Add(27 * time.Hour).Format(time.RFC3339), "1d3h"},
		{"past", now.Add(-10 * time.Minute).Format(time.RFC3339), "0m"},
		{"empty string", "", "0m"},
	}
	for _, tt := range tests {
		got := FormatResetTime(tt.resetISO, now)
		if got != tt.want {
			t.Errorf("FormatResetTime(%q) [%s] = %q, want %q", tt.resetISO, tt.name, got, tt.want)
		}
	}
}

func TestAbbreviatePath(t *testing.T) {
	tests := []struct {
		name       string
		cwd        string
		projectDir string
		home       string
		want       string
	}{
		{
			"at repo root",
			"/Users/t/Workspace/project",
			"/Users/t/Workspace/project",
			"/Users/t",
			"project",
		},
		{
			"inside repo subdirectory",
			"/Users/t/Workspace/project/src/pkg",
			"/Users/t/Workspace/project",
			"/Users/t",
			"project/src/pkg",
		},
		{
			"no project dir, fish-style",
			"/Users/t/Workspace/personal/foo",
			"",
			"/Users/t",
			"~/W/p/foo",
		},
		{
			"hidden dir, fish-style",
			"/Users/t/.config/nvim",
			"",
			"/Users/t",
			"~/.c/nvim",
		},
		{
			"short path, no abbreviation",
			"/tmp",
			"",
			"/Users/t",
			"/tmp",
		},
	}
	for _, tt := range tests {
		got := AbbreviatePath(tt.cwd, tt.projectDir, tt.home)
		if got != tt.want {
			t.Errorf("AbbreviatePath(%q, %q, %q) [%s] = %q, want %q",
				tt.cwd, tt.projectDir, tt.home, tt.name, got, tt.want)
		}
	}
}
