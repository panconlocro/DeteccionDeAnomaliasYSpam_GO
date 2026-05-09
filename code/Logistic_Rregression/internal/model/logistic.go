package model

import (
	"math"
	"sync"

	"detecciondeanomalias/code/Logistic_Rregression/internal/features"
)

type Example struct {
	X features.SparseVector
	Y int
}

type TrainConfig struct {
	Epochs       int
	LearningRate float64
	LambdaL2     float64
	BatchSize    int
	Threshold    float64
}

type LogisticRegression struct {
	Weights   []float64 `json:"-"`
	Bias      float64   `json:"bias"`
	Threshold float64   `json:"threshold"`
}

func Train(examples []Example, dimension int, cfg TrainConfig) *LogisticRegression {
	cfg = normalizeConfig(cfg, len(examples))
	m := NewLogisticRegression(dimension, cfg.Threshold)

	for epoch := 0; epoch < cfg.Epochs; epoch++ {
		for start := 0; start < len(examples); start += cfg.BatchSize {
			end := min(start+cfg.BatchSize, len(examples))
			grad := make([]float64, dimension)
			biasGrad := 0.0

			for i := start; i < end; i++ {
				err := m.PredictProba(examples[i].X) - float64(examples[i].Y)
				for _, f := range examples[i].X {
					if f.Index >= 0 && f.Index < dimension {
						grad[f.Index] += err * f.Value
					}
				}
				biasGrad += err
			}

			m.applyGradient(grad, biasGrad, end-start, cfg)
		}
	}

	return m
}

func TrainConcurrent(
	examples []Example,
	dimension int,
	cfg TrainConfig,
	workers int,
) *LogisticRegression {
	if workers <= 1 {
		return Train(examples, dimension, cfg)
	}

	cfg = normalizeConfig(cfg, len(examples))
	m := NewLogisticRegression(dimension, cfg.Threshold)

	for epoch := 0; epoch < cfg.Epochs; epoch++ {
		for start := 0; start < len(examples); start += cfg.BatchSize {
			end := min(start+cfg.BatchSize, len(examples))
			batchSize := end - start
			actualWorkers := min(workers, batchSize)
			chunkSize := (batchSize + actualWorkers - 1) / actualWorkers

			grads := make([][]float64, actualWorkers)
			biasGrads := make([]float64, actualWorkers)

			var wg sync.WaitGroup
			for worker := 0; worker < actualWorkers; worker++ {
				localStart := start + worker*chunkSize
				localEnd := min(localStart+chunkSize, end)
				if localStart >= localEnd {
					continue
				}

				grads[worker] = make([]float64, dimension)
				wg.Add(1)
				go func(worker, localStart, localEnd int) {
					defer wg.Done()

					localGrad := grads[worker]
					localBias := 0.0
					for i := localStart; i < localEnd; i++ {
						err := m.PredictProba(examples[i].X) - float64(examples[i].Y)
						for _, f := range examples[i].X {
							if f.Index >= 0 && f.Index < dimension {
								localGrad[f.Index] += err * f.Value
							}
						}
						localBias += err
					}
					biasGrads[worker] = localBias
				}(worker, localStart, localEnd)
			}

			wg.Wait()

			grad := make([]float64, dimension)
			biasGrad := 0.0
			for worker := 0; worker < actualWorkers; worker++ {
				if grads[worker] == nil {
					continue
				}
				for j := 0; j < dimension; j++ {
					grad[j] += grads[worker][j]
				}
				biasGrad += biasGrads[worker]
			}

			m.applyGradient(grad, biasGrad, batchSize, cfg)
		}
	}

	return m
}

func NewLogisticRegression(dimension int, threshold float64) *LogisticRegression {
	if threshold <= 0 || threshold >= 1 {
		threshold = 0.5
	}

	return &LogisticRegression{
		Weights:   make([]float64, dimension),
		Threshold: threshold,
	}
}

func (m *LogisticRegression) PredictProba(x features.SparseVector) float64 {
	z := m.Bias
	for _, f := range x {
		if f.Index >= 0 && f.Index < len(m.Weights) {
			z += m.Weights[f.Index] * f.Value
		}
	}
	return Sigmoid(z)
}

func (m *LogisticRegression) Predict(x features.SparseVector) int {
	if m.PredictProba(x) >= m.Threshold {
		return 1
	}
	return 0
}

func Sigmoid(z float64) float64 {
	if z >= 0 {
		expNeg := math.Exp(-z)
		return 1 / (1 + expNeg)
	}
	expZ := math.Exp(z)
	return expZ / (1 + expZ)
}

func (m *LogisticRegression) applyGradient(
	grad []float64,
	biasGrad float64,
	batchSize int,
	cfg TrainConfig,
) {
	scale := 1 / float64(batchSize)
	for j := range m.Weights {
		g := grad[j]*scale + cfg.LambdaL2*m.Weights[j]
		m.Weights[j] -= cfg.LearningRate * g
	}
	m.Bias -= cfg.LearningRate * biasGrad * scale
}

func normalizeConfig(cfg TrainConfig, n int) TrainConfig {
	if cfg.Epochs <= 0 {
		cfg.Epochs = 1
	}
	if cfg.LearningRate <= 0 {
		cfg.LearningRate = 0.05
	}
	if cfg.BatchSize <= 0 || cfg.BatchSize > n {
		cfg.BatchSize = max(1, n)
	}
	if cfg.Threshold <= 0 || cfg.Threshold >= 1 {
		cfg.Threshold = 0.5
	}
	return cfg
}
