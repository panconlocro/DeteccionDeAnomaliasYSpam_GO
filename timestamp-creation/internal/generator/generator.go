package generator

import (
	"fmt"
	"math/rand"
	"time"

	"timestamp-creation/internal/config"
	"timestamp-creation/internal/model"
	"timestamp-creation/internal/parse"
)

// PatternConfig defines weights and time-shape controls for synthetic patterns.
type PatternConfig struct {
	NormalRatio           float64
	BurstRatio            float64
	OffHoursRatio         float64
	CoordinatedRatio      float64
	RegularIntervalsRatio float64
	BurstWindowMinutes    int
}

// Generator creates synthetic timestamps and pattern metadata.
type Generator struct {
	cfg      PatternConfig
	location *time.Location
	rngs     RNGFactory
}

func New(cfg config.Config) (*Generator, error) {
	if cfg.Location == nil {
		return nil, fmt.Errorf("config location is nil; parse flags first")
	}

	return &Generator{
		cfg: PatternConfig{
			NormalRatio:           cfg.NormalRatio,
			BurstRatio:            cfg.BurstRatio,
			OffHoursRatio:         cfg.OffHoursRatio,
			CoordinatedRatio:      cfg.CoordinatedRatio,
			RegularIntervalsRatio: cfg.RegularIntervalsRatio,
			BurstWindowMinutes:    cfg.BurstWindowMinutes,
		},
		location: cfg.Location,
		rngs:     NewRNGFactory(cfg.Seed),
	}, nil
}

// WorkerRNG returns an RNG deterministic by worker id.
func (g *Generator) WorkerRNG(workerID int) *rand.Rand {
	return g.rngs.WorkerRNG(workerID)
}

// EnrichDateReceived returns the synthetic timestamp and associated pattern metadata.
func (g *Generator) EnrichDateReceived(
	row model.Row,
	rawDate string,
	key model.ProfileKey,
	profile model.GroupProfile,
) (string, model.PatternDecision, error) {
	parsed, err := parse.ParseDateReceived(rawDate, g.location)
	if err != nil {
		return "", model.PatternDecision{Type: model.PatternParseErrorFallback}, err
	}

	rowRNG := g.rngs.RowRNG(row.Index)
	decision := pickPattern(rowRNG, parsed.Day, profile, key, g.cfg)
	seconds := secondsForPattern(rowRNG, parsed.Day, decision, row.Index, g.cfg)

	ts, err := parse.ComposeTimestamp(parsed.Day, seconds, g.location)
	if err != nil {
		return "", decision, err
	}

	return ts.Format(time.RFC3339), decision, nil
}
