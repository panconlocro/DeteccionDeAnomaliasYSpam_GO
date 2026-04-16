package clean

import (
	"regexp"
	"strings"
)

var spaceRe = regexp.MustCompile(`\s+`)

// NormalizeText applies a simple deterministic normalization suitable for
// grouping repeated narratives across large CSVs.
func NormalizeText(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return ""
	}
	s = spaceRe.ReplaceAllString(s, " ")
	return s
}
