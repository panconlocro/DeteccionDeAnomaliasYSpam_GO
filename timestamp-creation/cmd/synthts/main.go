package main

import (
	"context"
	"fmt"
	"os"
	"sort"

	"timestamp-creation/internal/app"
	"timestamp-creation/internal/config"
	"timestamp-creation/internal/model"
)

func main() {
	cfg, err := config.Parse()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(2)
	}

	snapshot, err := app.Run(context.Background(), cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "run error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("rows_processed=%d parse_errors=%d\n", snapshot.RowsProcessed, snapshot.ParseErrors)
	for _, key := range sortedPatternKeys(snapshot.PatternCounts) {
		fmt.Printf("pattern_%s=%d\n", key, snapshot.PatternCounts[key])
	}
}

func sortedPatternKeys(m map[model.PatternType]int64) []model.PatternType {
	keys := make([]model.PatternType, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})
	return keys
}
