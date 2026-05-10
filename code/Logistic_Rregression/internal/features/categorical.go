package features

import (
	"sort"
	"strings"
)

type CategoryEncoder struct {
	Materia map[string]int
	Tipo    map[string]int
}

type categoryCount struct {
	Value string
	Count int
}

func BuildCategoryEncoder(materias, tipos []string, maxLevels int) CategoryEncoder {
	materiaCounts := make(map[string]int)
	tipoCounts := make(map[string]int)

	for _, value := range materias {
		value = NormalizeCategory(value)
		if value != "" {
			materiaCounts[value]++
		}
	}

	for _, value := range tipos {
		value = NormalizeCategory(value)
		if value != "" {
			tipoCounts[value]++
		}
	}

	return NewCategoryEncoderFromCounts(materiaCounts, tipoCounts, maxLevels)
}

func NewCategoryEncoderFromCounts(
	materiaCounts map[string]int,
	tipoCounts map[string]int,
	maxLevels int,
) CategoryEncoder {
	return CategoryEncoder{
		Materia: topCategoryMap(materiaCounts, maxLevels),
		Tipo:    topCategoryMap(tipoCounts, maxLevels),
	}
}

func (e CategoryEncoder) FeatureCount() int {
	return len(e.Materia) + len(e.Tipo)
}

func (e CategoryEncoder) Append(v SparseVector, baseIndex int, materia, tipo string) SparseVector {
	if idx, ok := e.Materia[NormalizeCategory(materia)]; ok {
		v = append(v, Feature{Index: baseIndex + idx, Value: 1})
	}

	tipoBase := baseIndex + len(e.Materia)
	if idx, ok := e.Tipo[NormalizeCategory(tipo)]; ok {
		v = append(v, Feature{Index: tipoBase + idx, Value: 1})
	}

	return v
}

func NormalizeCategory(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ToLower(value)
	value = removeAccents(value)
	value = strings.Join(strings.Fields(value), " ")
	return value
}

func topCategoryMap(counts map[string]int, maxLevels int) map[string]int {
	items := make([]categoryCount, 0, len(counts))
	for value, count := range counts {
		if value == "" {
			continue
		}
		items = append(items, categoryCount{Value: value, Count: count})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Count == items[j].Count {
			return items[i].Value < items[j].Value
		}
		return items[i].Count > items[j].Count
	})

	if maxLevels > 0 && len(items) > maxLevels {
		items = items[:maxLevels]
	}

	out := make(map[string]int, len(items))
	for i, item := range items {
		out[item.Value] = i
	}
	return out
}

func removeAccents(value string) string {
	replacer := strings.NewReplacer(
		"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u",
		"à", "a", "è", "e", "ì", "i", "ò", "o", "ù", "u",
		"ä", "a", "ë", "e", "ï", "i", "ö", "o", "ü", "u",
		"ñ", "n",
	)
	return replacer.Replace(value)
}
