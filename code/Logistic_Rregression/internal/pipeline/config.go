package pipeline

import "detecciondeanomalias/code/Logistic_Rregression/internal/benchmark"

type Config struct {
	Input        string
	Runs         int
	Epochs       int
	LearningRate float64
	LambdaL2     float64
	BatchSize    int
	MaxFeatures  int
	MinDF        int
	Threshold    float64
	Seed         int64
	TestRatio    float64
	UseBigrams   bool
	Limit        int
}

func (c Config) normalize() Config {
	if c.Runs <= 0 {
		c.Runs = 1
	}
	if c.Epochs <= 0 {
		c.Epochs = 10
	}
	if c.LearningRate <= 0 {
		c.LearningRate = 0.05
	}
	if c.BatchSize <= 0 {
		c.BatchSize = 1024
	}
	if c.MaxFeatures <= 0 {
		c.MaxFeatures = 5000
	}
	if c.MinDF <= 0 {
		c.MinDF = 2
	}
	if c.Threshold <= 0 || c.Threshold >= 1 {
		c.Threshold = 0.5
	}
	if c.TestRatio <= 0 || c.TestRatio >= 1 {
		c.TestRatio = 0.2
	}
	return c
}

func (c Config) benchmarkConfig() benchmark.ModelConfig {
	c = c.normalize()
	return benchmark.ModelConfig{
		Epochs:       c.Epochs,
		LearningRate: c.LearningRate,
		LambdaL2:     c.LambdaL2,
		BatchSize:    c.BatchSize,
		MaxFeatures:  c.MaxFeatures,
		MinDF:        c.MinDF,
		Threshold:    c.Threshold,
		Seed:         c.Seed,
		UseBigrams:   c.UseBigrams,
		TestRatio:    c.TestRatio,
	}
}
