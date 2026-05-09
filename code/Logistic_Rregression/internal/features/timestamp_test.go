package features

import "testing"

func TestParseTimestampDirect(t *testing.T) {
	ts, ok := ParseTimestamp("2026-05-09 22:15:00", "", "")
	if !ok {
		t.Fatal("expected timestamp to parse")
	}
	if ts.Hour() != 22 || ts.Month() != 5 {
		t.Fatalf("unexpected timestamp: %v", ts)
	}
}

func TestParseTimestampFromDateAndHour(t *testing.T) {
	ts, ok := ParseTimestamp("", "2026-05-09", "08:30")
	if !ok {
		t.Fatal("expected date/hour to parse")
	}
	if ts.Hour() != 8 || ts.Minute() != 30 {
		t.Fatalf("unexpected timestamp: %v", ts)
	}
}

func TestTimestampFeatures(t *testing.T) {
	ts, ok := ParseTimestamp("2026-05-09 02:15:00", "", "")
	values := TimestampFeatures(ts, ok)
	if len(values) != TimestampFeatureCount {
		t.Fatalf("feature count = %d, want %d", len(values), TimestampFeatureCount)
	}
	if values[3] != 1 {
		t.Fatalf("expected weekend feature to be 1, got %v", values[3])
	}
	if values[5] != 1 {
		t.Fatalf("expected night feature to be 1, got %v", values[5])
	}
}
