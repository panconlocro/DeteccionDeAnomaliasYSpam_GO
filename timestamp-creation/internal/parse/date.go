package parse

import (
	"fmt"
	"strings"
	"time"

	"timestamp-creation/internal/model"
)

var dateOnlyLayouts = []string{
	"2006-01-02", // YYYY-MM-DD
	"01/02/06",   // MM/DD/YY
	"01/02/2006", // MM/DD/YYYY
	"1/2/06",     // M/D/YY
	"1/2/2006",   // M/D/YYYY
}

var timestampLayouts = []string{
	time.RFC3339Nano,
	time.RFC3339,
}

// ParseDateReceived accepts YYYY-MM-DD, MM/DD/YY, MM/DD/YYYY and RFC3339-compatible values.
// It always returns the day normalized to midnight in loc while preserving
// the original calendar date found in the input value.
func ParseDateReceived(raw string, loc *time.Location) (model.DateParseResult, error) {
	if loc == nil {
		loc = time.UTC
	}

	value := strings.TrimSpace(raw)
	if value == "" {
		return model.DateParseResult{}, fmt.Errorf("empty Date received value")
	}

	for _, layout := range dateOnlyLayouts {
		if t, err := time.ParseInLocation(layout, value, loc); err == nil {
			return model.DateParseResult{
				Day:          startOfDay(t, loc),
				HadTimestamp: false,
				LayoutUsed:   layout,
			}, nil
		}
	}

	for _, layout := range timestampLayouts {
		t, err := time.Parse(layout, value)
		if err != nil {
			continue
		}

		y, m, d := t.Date()
		day := time.Date(y, m, d, 0, 0, 0, 0, loc)
		return model.DateParseResult{
			Day:          day,
			HadTimestamp: true,
			LayoutUsed:   layout,
		}, nil
	}

	return model.DateParseResult{}, fmt.Errorf(
		"unsupported Date received format %q (expected YYYY-MM-DD, MM/DD/YY, MM/DD/YYYY, or RFC3339)",
		value,
	)
}

// ComposeTimestamp builds a synthetic timestamp from a day and second offset.
func ComposeTimestamp(day time.Time, secondsSinceMidnight int, loc *time.Location) (time.Time, error) {
	if loc == nil {
		loc = time.UTC
	}
	if secondsSinceMidnight < 0 || secondsSinceMidnight > 86399 {
		return time.Time{}, fmt.Errorf("secondsSinceMidnight must be in [0, 86399], got %d", secondsSinceMidnight)
	}

	start := startOfDay(day, loc)
	return start.Add(time.Duration(secondsSinceMidnight) * time.Second), nil
}

// DayBounds returns [start, end] for the day of the given time in loc.
func DayBounds(day time.Time, loc *time.Location) (time.Time, time.Time) {
	if loc == nil {
		loc = time.UTC
	}
	start := startOfDay(day, loc)
	end := start.Add(24*time.Hour - time.Nanosecond)
	return start, end
}

func startOfDay(t time.Time, loc *time.Location) time.Time {
	y, m, d := t.In(loc).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, loc)
}
