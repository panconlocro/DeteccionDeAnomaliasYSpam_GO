package data

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
)

func Split(records []Record, testRatio float64, seed int64) ([]Record, []Record, string, error) {
	if len(records) < 2 {
		return nil, nil, "", fmt.Errorf("se requieren al menos 2 registros para train/test")
	}

	if testRatio <= 0 || testRatio >= 1 {
		testRatio = 0.2
	}

	ordered := make([]Record, len(records))
	copy(ordered, records)

	method := "temporal"
	for _, record := range ordered {
		if !record.HasTimestamp {
			method = "random"
			break
		}
	}

	if method == "temporal" {
		sort.SliceStable(ordered, func(i, j int) bool {
			if ordered[i].Timestamp.Equal(ordered[j].Timestamp) {
				return ordered[i].Index < ordered[j].Index
			}
			return ordered[i].Timestamp.Before(ordered[j].Timestamp)
		})
	} else {
		rng := rand.New(rand.NewSource(seed))
		rng.Shuffle(len(ordered), func(i, j int) {
			ordered[i], ordered[j] = ordered[j], ordered[i]
		})
	}

	testSize := int(math.Round(float64(len(ordered)) * testRatio))
	if testSize < 1 {
		testSize = 1
	}
	if testSize >= len(ordered) {
		testSize = len(ordered) - 1
	}

	splitAt := len(ordered) - testSize
	train := append([]Record(nil), ordered[:splitAt]...)
	test := append([]Record(nil), ordered[splitAt:]...)

	return train, test, method, nil
}
