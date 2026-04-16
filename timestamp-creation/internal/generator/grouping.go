package generator

import (
	"encoding/hex"
	"hash/fnv"
	"strings"

	"timestamp-creation/internal/clean"
	"timestamp-creation/internal/model"
)

const emptyNarrativeSentinel = "<empty-narrative>"

// BuildProfileKey creates the grouping key from normalized textual signals.
func BuildProfileKey(record []string, cols model.RequiredColumns) model.ProfileKey {
	narrative := field(record, cols.Narrative)
	company := field(record, cols.Company)
	product := field(record, cols.Product)
	issue := field(record, cols.Issue)

	normNarrative := clean.NormalizeText(narrative)
	if normNarrative == "" {
		normNarrative = emptyNarrativeSentinel
	}

	return model.ProfileKey{
		NarrativeHash: HashNarrative(normNarrative),
		Company:       clean.NormalizeText(company),
		Product:       clean.NormalizeText(product),
		Issue:         clean.NormalizeText(issue),
	}
}

// HashNarrative computes a stable compact hash used for grouping.
func HashNarrative(normalizedNarrative string) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(normalizedNarrative))
	sum := h.Sum(nil)
	return hex.EncodeToString(sum)
}

// BuildCampaignID creates a deterministic campaign identifier.
func BuildCampaignID(key model.ProfileKey) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(strings.Join([]string{
		key.NarrativeHash,
		key.Company,
		key.Product,
		key.Issue,
	}, "|")))
	return hex.EncodeToString(h.Sum(nil))[:12]
}

func field(record []string, idx int) string {
	if idx < 0 || idx >= len(record) {
		return ""
	}
	return record[idx]
}
