package model

import "time"

// RequiredColumns stores indices for the minimum columns needed by the pipeline.
type RequiredColumns struct {
	DateReceived int
	Narrative    int
	Company      int
	Product      int
	Issue        int
	State        int
}

// ColumnIndexes provides direct access to required columns and a full map by name.
type ColumnIndexes struct {
	Required RequiredColumns
	ByName   map[string]int
}

// Row is a CSV row flowing through the pipeline with its original index.
type Row struct {
	Index  int
	Record []string
}

// ProfileKey identifies repetition/campaign candidates in the profiling phase.
type ProfileKey struct {
	NarrativeHash string
	Company       string
	Product       string
	Issue         string
}

// GroupProfile stores aggregated information for a key discovered in pass 1.
type GroupProfile struct {
	Count  int
	States map[string]int
}

// PatternType classifies the synthetic behavior assigned to a row.
type PatternType string

const (
	PatternNormal             PatternType = "normal"
	PatternBurst              PatternType = "burst"
	PatternOffHours           PatternType = "offhours"
	PatternRegularIntervals   PatternType = "regular_intervals"
	PatternCoordinated        PatternType = "coordinated"
	PatternParseErrorFallback PatternType = "parse_error_fallback"
)

// PatternDecision carries metadata used to annotate optional output columns.
type PatternDecision struct {
	Type             PatternType
	CampaignID       string
	SeededSuspicious bool
}

// ProcessedRow is the worker output before ordered writing.
type ProcessedRow struct {
	Index    int
	Record   []string
	Decision PatternDecision
	ParseErr error
}

// DateParseResult keeps normalized day information from Date received.
type DateParseResult struct {
	Day          time.Time
	HadTimestamp bool
	LayoutUsed   string
}
