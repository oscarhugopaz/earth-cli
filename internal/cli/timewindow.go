package cli

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const dateLayout = "2006-01-02"

// parseTimeWindow turns --from/--to/--since into an absolute window.
//
// --since is relative to now and cannot be combined with --from. Date-only
// --from is inclusive at 00:00:00Z; date-only --to is inclusive at 23:59:59Z.
func parseTimeWindow(from, to, since string, now time.Time) (start, end *time.Time, label string, err error) {
	from = strings.TrimSpace(from)
	to = strings.TrimSpace(to)
	since = strings.TrimSpace(since)

	if since != "" && from != "" {
		return nil, nil, "", fmt.Errorf("--since cannot be combined with --from")
	}

	if since != "" {
		duration, parseErr := parseSince(since)
		if parseErr != nil {
			return nil, nil, "", parseErr
		}
		computedStart := now.Add(-duration)
		computedEnd := now
		return &computedStart, &computedEnd, humanizeSince(duration), nil
	}

	if from != "" {
		parsed, parseErr := parseDateStart(from)
		if parseErr != nil {
			return nil, nil, "", parseErr
		}
		start = &parsed
	}
	if to != "" {
		parsed, parseErr := parseDateEnd(to)
		if parseErr != nil {
			return nil, nil, "", parseErr
		}
		end = &parsed
	}

	if start != nil && end != nil && end.Before(*start) {
		return nil, nil, "", fmt.Errorf("--to (%s) must not be before --from (%s)", to, from)
	}

	switch {
	case start != nil && end != nil:
		label = start.Format(dateLayout) + " to " + end.Format(dateLayout)
	case start != nil:
		label = "since " + start.Format(dateLayout)
	case end != nil:
		label = "until " + end.Format(dateLayout)
	}

	return start, end, label, nil
}

func parseSince(raw string) (time.Duration, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, fmt.Errorf("invalid --since: duration is empty")
	}

	last := trimmed[len(trimmed)-1]
	if last == 'd' || last == 'w' {
		value, err := strconv.ParseFloat(trimmed[:len(trimmed)-1], 64)
		if err != nil || value <= 0 {
			return 0, fmt.Errorf("invalid --since %q: use a duration like 30d, 12h or 2w", raw)
		}
		unit := 24 * time.Hour
		if last == 'w' {
			unit = 7 * 24 * time.Hour
		}
		return time.Duration(value * float64(unit)), nil
	}

	duration, err := time.ParseDuration(trimmed)
	if err != nil || duration <= 0 {
		return 0, fmt.Errorf("invalid --since %q: use a duration like 30d, 12h or 2w", raw)
	}
	return duration, nil
}

func parseDateStart(raw string) (time.Time, error) {
	if parsed, err := time.Parse(dateLayout, raw); err == nil {
		return parsed.UTC(), nil
	}
	if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
		return parsed.UTC(), nil
	}
	return time.Time{}, fmt.Errorf("invalid date %q: use YYYY-MM-DD or RFC3339", raw)
}

func parseDateEnd(raw string) (time.Time, error) {
	if parsed, err := time.Parse(dateLayout, raw); err == nil {
		return parsed.Add(24*time.Hour - time.Second).UTC(), nil
	}
	if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
		return parsed.UTC(), nil
	}
	return time.Time{}, fmt.Errorf("invalid date %q: use YYYY-MM-DD or RFC3339", raw)
}

func humanizeSince(duration time.Duration) string {
	switch {
	case duration%(24*time.Hour) == 0:
		days := int(duration / (24 * time.Hour))
		return fmt.Sprintf("last %d day%s", days, plural(days))
	case duration%time.Hour == 0:
		hours := int(duration / time.Hour)
		return fmt.Sprintf("last %d hour%s", hours, plural(hours))
	default:
		return "last " + duration.String()
	}
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func nowUTC() time.Time {
	return time.Now().UTC()
}
