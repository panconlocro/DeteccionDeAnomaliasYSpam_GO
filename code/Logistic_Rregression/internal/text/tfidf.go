package text

import (
	"math"
	"sort"

	"detecciondeanomalias/code/Logistic_Rregression/internal/features"
)

type Vectorizer struct {
	Vocab map[string]int
	IDF   []float64
}

type dfItem struct {
	Token string
	DF    int
}

func BuildVectorizer(docs [][]string, maxFeatures, minDF int) *Vectorizer {
	return NewVectorizerFromDF(CountDocumentFrequency(docs), len(docs), maxFeatures, minDF)
}

func CountDocumentFrequency(docs [][]string) map[string]int {
	df := make(map[string]int)
	for _, tokens := range docs {
		seen := make(map[string]struct{}, len(tokens))
		for _, token := range tokens {
			if token == "" {
				continue
			}
			seen[token] = struct{}{}
		}
		for token := range seen {
			df[token]++
		}
	}
	return df
}

func NewVectorizerFromDF(
	df map[string]int,
	documentCount int,
	maxFeatures int,
	minDF int,
) *Vectorizer {
	if minDF <= 0 {
		minDF = 1
	}

	items := make([]dfItem, 0, len(df))
	for token, count := range df {
		if count >= minDF {
			items = append(items, dfItem{Token: token, DF: count})
		}
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].DF == items[j].DF {
			return items[i].Token < items[j].Token
		}
		return items[i].DF > items[j].DF
	})

	if maxFeatures > 0 && len(items) > maxFeatures {
		items = items[:maxFeatures]
	}

	vocab := make(map[string]int, len(items))
	idf := make([]float64, len(items))
	for i, item := range items {
		vocab[item.Token] = i
		idf[i] = math.Log((1+float64(documentCount))/(1+float64(item.DF))) + 1
	}

	return &Vectorizer{
		Vocab: vocab,
		IDF:   idf,
	}
}

func (v *Vectorizer) FeatureCount() int {
	if v == nil {
		return 0
	}
	return len(v.IDF)
}

func (v *Vectorizer) Vectorize(tokens []string) features.SparseVector {
	if v == nil || len(tokens) == 0 {
		return nil
	}

	counts := make(map[int]int)
	for _, token := range tokens {
		if idx, ok := v.Vocab[token]; ok {
			counts[idx]++
		}
	}

	if len(counts) == 0 {
		return nil
	}

	out := make(features.SparseVector, 0, len(counts))
	tokenCount := float64(len(tokens))
	for idx, count := range counts {
		out = append(out, features.Feature{
			Index: idx,
			Value: (float64(count) / tokenCount) * v.IDF[idx],
		})
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Index < out[j].Index
	})

	return out
}
