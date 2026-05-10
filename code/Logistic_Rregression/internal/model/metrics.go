package model

import (
	"sync"
)

type Metrics struct {
	Accuracy  float64 `json:"accuracy"`
	Precision float64 `json:"precision"`
	Recall    float64 `json:"recall"`
	F1        float64 `json:"f1"`
	TP        int     `json:"tp"`
	TN        int     `json:"tn"`
	FP        int     `json:"fp"`
	FN        int     `json:"fn"`
}

func Evaluate(m *LogisticRegression, examples []Example) Metrics {
	var out Metrics
	for _, example := range examples {
		out.Add(example.Y, m.Predict(example.X))
	}
	return out.Finalize()
}

func EvaluateConcurrent(m *LogisticRegression, examples []Example, workers int) Metrics {
	if workers <= 1 || len(examples) < 2 {
		return Evaluate(m, examples)
	}

	actualWorkers := min(workers, len(examples))
	chunkSize := (len(examples) + actualWorkers - 1) / actualWorkers

	var out Metrics
	var mu sync.Mutex
	var wg sync.WaitGroup
	for worker := 0; worker < actualWorkers; worker++ {
		start := worker * chunkSize
		end := min(start+chunkSize, len(examples))
		if start >= end {
			continue
		}

		wg.Add(1)
		go func(worker, start, end int) {
			defer wg.Done()

			var local Metrics
			for i := start; i < end; i++ {
				local.Add(examples[i].Y, m.Predict(examples[i].X))
			}

			mu.Lock()
			out.TP += local.TP
			out.TN += local.TN
			out.FP += local.FP
			out.FN += local.FN
			mu.Unlock()
		}(worker, start, end)
	}

	wg.Wait()

	return out.Finalize()
}

func (m *Metrics) Add(actual int, predicted int) {
	switch {
	case actual == 1 && predicted == 1:
		m.TP++
	case actual == 0 && predicted == 0:
		m.TN++
	case actual == 0 && predicted == 1:
		m.FP++
	case actual == 1 && predicted == 0:
		m.FN++
	}
}

func (m Metrics) Finalize() Metrics {
	total := m.TP + m.TN + m.FP + m.FN
	if total > 0 {
		m.Accuracy = float64(m.TP+m.TN) / float64(total)
	}

	if m.TP+m.FP > 0 {
		m.Precision = float64(m.TP) / float64(m.TP+m.FP)
	}

	if m.TP+m.FN > 0 {
		m.Recall = float64(m.TP) / float64(m.TP+m.FN)
	}

	if m.Precision+m.Recall > 0 {
		m.F1 = 2 * m.Precision * m.Recall / (m.Precision + m.Recall)
	}

	return m
}
