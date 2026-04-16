package config

import (
	"flag"
	"fmt"
	"os"
	"time"
)

// Config centralizes runtime parameters for reproducible synthetic timestamp generation.
type Config struct {
	InputPath  string
	OutputPath string

	Workers int
	Seed    int64

	NormalRatio           float64
	BurstRatio            float64
	OffHoursRatio         float64
	CoordinatedRatio      float64
	RegularIntervalsRatio float64

	BurstWindowMinutes int
	AddPatternColumns  bool

	Timezone string
	Location *time.Location

	ChannelBuffer int
}

// Parse reads flags from os.Args.
func Parse() (Config, error) {
	return ParseArgs(os.Args[1:])
}

// ParseArgs reads and validates all CLI flags.
func ParseArgs(args []string) (Config, error) {
	fs := flag.NewFlagSet("synthts", flag.ContinueOnError)

	cfg := Config{}

	fs.StringVar(&cfg.InputPath, "input", "", "input CSV path")
	fs.StringVar(&cfg.OutputPath, "output", "", "output CSV path")

	fs.IntVar(&cfg.Workers, "workers", 8, "number of worker goroutines")
	fs.Int64Var(&cfg.Seed, "seed", 42, "master seed for reproducible generation")

	fs.Float64Var(&cfg.NormalRatio, "normal-ratio", 0.85, "base probability of normal behavior")
	fs.Float64Var(&cfg.BurstRatio, "burst-ratio", 0.07, "probability weight for burst suspicious pattern")
	fs.Float64Var(&cfg.OffHoursRatio, "offhours-ratio", 0.04, "probability weight for off-hours suspicious pattern")
	fs.Float64Var(&cfg.CoordinatedRatio, "coordinated-ratio", 0.03, "probability weight for coordinated suspicious pattern")
	fs.Float64Var(&cfg.RegularIntervalsRatio, "regular-intervals-ratio", 0.01, "probability weight for near-regular intervals suspicious pattern")

	fs.IntVar(&cfg.BurstWindowMinutes, "burst-window-minutes", 10, "max minutes for burst windows")
	fs.BoolVar(&cfg.AddPatternColumns, "add-pattern-columns", true, "add synthetic_pattern_type, synthetic_campaign_id and synthetic_is_seeded_suspicious")

	fs.StringVar(&cfg.Timezone, "timezone", "UTC", "IANA timezone for synthetic timestamps, e.g. America/New_York")
	fs.IntVar(&cfg.ChannelBuffer, "channel-buffer", 2048, "channel buffer size for pipeline stages")

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// SuspiciousWeight returns the total suspicious weight configured.
func (c Config) SuspiciousWeight() float64 {
	return c.BurstRatio + c.OffHoursRatio + c.CoordinatedRatio + c.RegularIntervalsRatio
}

func (c *Config) validate() error {
	if c.InputPath == "" {
		return fmt.Errorf("missing required flag --input")
	}
	if c.OutputPath == "" {
		return fmt.Errorf("missing required flag --output")
	}
	if c.Workers <= 0 {
		return fmt.Errorf("--workers must be > 0")
	}
	if c.ChannelBuffer <= 0 {
		return fmt.Errorf("--channel-buffer must be > 0")
	}
	if c.BurstWindowMinutes <= 0 || c.BurstWindowMinutes > 24*60 {
		return fmt.Errorf("--burst-window-minutes must be between 1 and 1440")
	}

	ratios := map[string]float64{
		"--normal-ratio":            c.NormalRatio,
		"--burst-ratio":             c.BurstRatio,
		"--offhours-ratio":          c.OffHoursRatio,
		"--coordinated-ratio":       c.CoordinatedRatio,
		"--regular-intervals-ratio": c.RegularIntervalsRatio,
	}
	for name, value := range ratios {
		if value < 0 || value > 1 {
			return fmt.Errorf("%s must be in [0,1]", name)
		}
	}

	total := c.NormalRatio + c.SuspiciousWeight()
	if total <= 0 {
		return fmt.Errorf("sum of all ratios must be > 0")
	}
	if total > 1.0000001 {
		return fmt.Errorf("normal+suspicious ratios must be <= 1.0, got %.4f", total)
	}

	loc, err := time.LoadLocation(c.Timezone)
	if err != nil {
		return fmt.Errorf("invalid --timezone %q: %w", c.Timezone, err)
	}
	c.Location = loc

	return nil
}
