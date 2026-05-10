package features

import (
	"math"
	"strconv"
	"strings"
	"time"
)

const TimestampFeatureCount = 10

var timestampLayouts = []string{
	"2006-01-02 15:04:05",
	"2006-01-02 15:04",
	time.RFC3339,
	"2006-01-02",
	"02/01/2006 15:04:05",
	"02/01/2006 15:04",
	"02/01/2006",
	"2006/01/02 15:04:05",
	"2006/01/02 15:04",
	"2006/01/02",
}

var dateLayouts = []string{
	"2006-01-02",
	"02/01/2006",
	"2006/01/02",
}

func ParseTimestamp(timestampValue, dateValue, hourValue string) (time.Time, bool) {
	if t, ok := parseWithLayouts(timestampValue, timestampLayouts); ok {
		return t, true
	}

	date, ok := parseWithLayouts(dateValue, dateLayouts)
	if !ok {
		return time.Time{}, false
	}

	hour, minute, second, ok := parseClock(hourValue)
	if !ok {
		return date, true
	}

	return time.Date(
		date.Year(),
		date.Month(),
		date.Day(),
		hour,
		minute,
		second,
		0,
		time.UTC,
	), true
}

func parseWithLayouts(value string, layouts []string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, value); err == nil {
			return t, true
		}
	}

	return time.Time{}, false
}

func parseClock(value string) (int, int, int, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, 0, 0, false
	}

	parts := strings.Split(value, ":")
	if len(parts) < 2 {
		return 0, 0, 0, false
	}

	hour, err := strconv.Atoi(parts[0])
	if err != nil || hour < 0 || hour > 23 {
		return 0, 0, 0, false
	}

	minute, err := strconv.Atoi(parts[1])
	if err != nil || minute < 0 || minute > 59 {
		return 0, 0, 0, false
	}

	second := 0
	if len(parts) >= 3 {
		second, err = strconv.Atoi(parts[2])
		if err != nil || second < 0 || second > 59 {
			return 0, 0, 0, false
		}
	}

	return hour, minute, second, true
}

func TimestampFeatures(t time.Time, ok bool) [TimestampFeatureCount]float64 {
	var out [TimestampFeatureCount]float64
	if !ok {
		return out
	}

	hour := t.Hour()
	day := int(t.Weekday())
	month := int(t.Month())
	isWeekend := day == int(time.Saturday) || day == int(time.Sunday)
	isBusinessHour := !isWeekend && hour >= 8 && hour < 18
	isNight := hour < 5 || hour >= 22

	out[0] = float64(hour) / 23.0
	out[1] = float64(day) / 6.0
	out[2] = float64(month) / 12.0
	out[3] = boolFloat(isWeekend)
	out[4] = boolFloat(isBusinessHour)
	out[5] = boolFloat(isNight)
	out[6] = math.Sin(2 * math.Pi * float64(hour) / 24.0)
	out[7] = math.Cos(2 * math.Pi * float64(hour) / 24.0)
	out[8] = math.Sin(2 * math.Pi * float64(day) / 7.0)
	out[9] = math.Cos(2 * math.Pi * float64(day) / 7.0)

	return out
}

func boolFloat(v bool) float64 {
	if v {
		return 1
	}
	return 0
}
