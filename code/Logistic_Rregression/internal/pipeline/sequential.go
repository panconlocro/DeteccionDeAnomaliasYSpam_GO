package pipeline

import (
	"detecciondeanomalias/code/Logistic_Rregression/internal/benchmark"
)

func RunSequential(cfg Config) (benchmark.Report, error) {
	cfg = cfg.normalize()
	times := make([]float64, 0, cfg.Runs)
	stages := make([]benchmark.StageTimes, 0, cfg.Runs)

	var last runOutput
	for run := 0; run < cfg.Runs; run++ {
		out, err := runOnce(cfg, 1, false)
		if err != nil {
			return benchmark.Report{}, err
		}
		last = out
		times = append(times, out.StageTimes.Total)
		stages = append(stages, out.StageTimes)
	}

	result := benchmark.WorkerResult{
		Workers:            1,
		TimesSeconds:       times,
		AvgSeconds:         benchmark.Average(times),
		TrimmedMeanSeconds: benchmark.TrimmedMean(times),
		Metrics:            last.Metrics,
		StageTimes:         benchmark.AverageStageTimes(stages),
		SplitMethod:        last.SplitMethod,
	}

	return benchmark.Report{
		Mode:             "sequential",
		DatasetRows:      last.DatasetRows,
		TrainRows:        last.TrainRows,
		TestRows:         last.TestRows,
		Runs:             cfg.Runs,
		Config:           cfg.benchmarkConfig(),
		ResultsByWorkers: []benchmark.WorkerResult{result},
	}, nil
}
