package cli

import (
	"testing"
	"time"
)

func TestParseTimeWindowSince(t *testing.T) {
	now := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)

	start, end, label, err := parseTimeWindow("", "", "30d", now)
	if err != nil {
		t.Fatalf("parseTimeWindow returned error: %v", err)
	}
	if label != "last 30 days" {
		t.Fatalf("label = %q", label)
	}
	if start == nil || !start.Equal(now.Add(-30*24*time.Hour)) {
		t.Fatalf("start = %v", start)
	}
	if end == nil || !end.Equal(now) {
		t.Fatalf("end = %v", end)
	}
}

func TestParseTimeWindowHours(t *testing.T) {
	now := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	start, _, label, err := parseTimeWindow("", "", "12h", now)
	if err != nil {
		t.Fatalf("parseTimeWindow returned error: %v", err)
	}
	if label != "last 12 hours" {
		t.Fatalf("label = %q", label)
	}
	if start == nil || !start.Equal(now.Add(-12*time.Hour)) {
		t.Fatalf("start = %v", start)
	}
}

func TestParseTimeWindowDates(t *testing.T) {
	now := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	start, end, label, err := parseTimeWindow("2026-09-01", "2026-10-01", "", now)
	if err != nil {
		t.Fatalf("parseTimeWindow returned error: %v", err)
	}
	if label != "2026-09-01 to 2026-10-01" {
		t.Fatalf("label = %q", label)
	}
	wantStart := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	if start == nil || !start.Equal(wantStart) {
		t.Fatalf("start = %v, want %v", start, wantStart)
	}
	wantEnd := time.Date(2026, 10, 1, 23, 59, 59, 0, time.UTC)
	if end == nil || !end.Equal(wantEnd) {
		t.Fatalf("end = %v, want %v", end, wantEnd)
	}
}

func TestParseTimeWindowInvalid(t *testing.T) {
	now := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name            string
		from, to, since string
	}{
		{"since and from", "2026-09-01", "", "30d"},
		{"bad since", "", "", "soon"},
		{"zero since", "", "", "0d"},
		{"bad from", "yesterday", "", ""},
		{"to before from", "2026-10-01", "2026-09-01", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, _, err := parseTimeWindow(tc.from, tc.to, tc.since, now); err == nil {
				t.Fatalf("parseTimeWindow(%q,%q,%q) = nil error, want error", tc.from, tc.to, tc.since)
			}
		})
	}
}
