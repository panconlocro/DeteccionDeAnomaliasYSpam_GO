package csvio

import (
	"fmt"
	"strings"
	"unicode"

	"timestamp-creation/internal/model"
)

const (
	colSyntheticTS          = "synthetic_date_received_ts"
	colSyntheticPatternType = "synthetic_pattern_type"
	colSyntheticCampaignID  = "synthetic_campaign_id"
	colSyntheticIsSeeded    = "synthetic_is_seeded_suspicious"
)

var requiredCanonical = map[string]string{
	"datereceived":               "Date received",
	"consumercomplaintnarrative": "Consumer complaint narrative",
	"company":                    "Company",
	"product":                    "Product",
	"issue":                      "Issue",
	"state":                      "State",
}

func DetectColumnIndexes(header []string) (model.ColumnIndexes, error) {
	if len(header) == 0 {
		return model.ColumnIndexes{}, fmt.Errorf("empty CSV header")
	}

	byCanonical := make(map[string]int, len(header))
	byName := make(map[string]int, len(header))
	for i, h := range header {
		canon := canonicalHeader(h)
		if _, exists := byCanonical[canon]; !exists {
			byCanonical[canon] = i
		}
		byName[h] = i
	}

	missing := make([]string, 0)
	get := func(canon string) int {
		idx, ok := byCanonical[canon]
		if !ok {
			missing = append(missing, requiredCanonical[canon])
			return -1
		}
		return idx
	}

	required := model.RequiredColumns{
		DateReceived: get("datereceived"),
		Narrative:    get("consumercomplaintnarrative"),
		Company:      get("company"),
		Product:      get("product"),
		Issue:        get("issue"),
		State:        get("state"),
	}

	if len(missing) > 0 {
		return model.ColumnIndexes{}, fmt.Errorf("missing required columns: %s", strings.Join(missing, ", "))
	}

	return model.ColumnIndexes{Required: required, ByName: byName}, nil
}

func BuildOutputHeader(inputHeader []string, addPatternColumns bool) []string {
	out := make([]string, 0, len(inputHeader)+4)
	out = append(out, inputHeader...)
	out = append(out, colSyntheticTS)
	if addPatternColumns {
		out = append(out, colSyntheticPatternType, colSyntheticCampaignID, colSyntheticIsSeeded)
	}
	return out
}

func AppendSyntheticColumns(record []string, ts string, addPatternColumns bool, patternType, campaignID, seeded string) []string {
	out := make([]string, 0, len(record)+4)
	out = append(out, record...)
	out = append(out, ts)
	if addPatternColumns {
		out = append(out, patternType, campaignID, seeded)
	}
	return out
}

func canonicalHeader(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(name))
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}
