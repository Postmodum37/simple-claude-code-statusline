package main

import "testing"

func TestModelDisplayName(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		displayName string
		want        string
	}{
		{
			name: "new format opus with minor version",
			id:   "claude-opus-4-6",
			want: "Opus 4.6",
		},
		{
			name: "new format opus with minor and date",
			id:   "claude-opus-4-5-20251101",
			want: "Opus 4.5",
		},
		{
			name: "new format sonnet with minor version",
			id:   "claude-sonnet-4-6",
			want: "Sonnet 4.6",
		},
		{
			name: "new format sonnet major only with date",
			id:   "claude-sonnet-4-20250514",
			want: "Sonnet 4",
		},
		{
			name: "new format haiku with minor and date",
			id:   "claude-haiku-4-5-20251001",
			want: "Haiku 4.5",
		},
		{
			name: "old format with major.minor before model name",
			id:   "claude-3-5-sonnet-20241022",
			want: "Sonnet 3.5",
		},
		{
			name: "strip [1m] context suffix",
			id:   "claude-opus-4-6[1m]",
			want: "Opus 4.6",
		},
		{
			name: "empty id falls back to first word of display name",
			id:          "",
			displayName: "Claude 4 Opus",
			want:        "Claude",
		},
		{
			name:        "empty id and empty display name",
			id:          "",
			displayName: "",
			want:        "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ModelDisplayName(tt.id, tt.displayName)
			if got != tt.want {
				t.Errorf("ModelDisplayName(%q, %q) = %q, want %q", tt.id, tt.displayName, got, tt.want)
			}
		})
	}
}
