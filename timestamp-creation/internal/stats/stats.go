package stats

import (
	"sync"

	"timestamp-creation/internal/model"
)

type Snapshot struct {
	RowsProcessed int64
	ParseErrors   int64
	PatternCounts map[model.PatternType]int64
}

type Collector struct {
	mu sync.Mutex

	rowsProcessed int64
	parseErrors   int64
	patternCounts map[model.PatternType]int64
}

func NewCollector() *Collector {
	return &Collector{
		patternCounts: make(map[model.PatternType]int64),
	}
}

func (c *Collector) Observe(decision model.PatternDecision, parseErr error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.rowsProcessed++
	if parseErr != nil {
		c.parseErrors++
	}
	c.patternCounts[decision.Type]++
}

func (c *Collector) Snapshot() Snapshot {
	c.mu.Lock()
	defer c.mu.Unlock()

	copied := make(map[model.PatternType]int64, len(c.patternCounts))
	for k, v := range c.patternCounts {
		copied[k] = v
	}

	return Snapshot{
		RowsProcessed: c.rowsProcessed,
		ParseErrors:   c.parseErrors,
		PatternCounts: copied,
	}
}
