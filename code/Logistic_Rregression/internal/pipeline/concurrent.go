package pipeline

import (
	"fmt"
	"os"

	"detecciondeanomalias/code/Logistic_Rregression/internal/benchmark"
)

var concurrentSections = []string{
	"preprocess/tokenization",
	"document-frequency local maps + merge",
	"tf-idf vectorization",
	"mini-batch gradient calculation + reduce",
	"evaluation/prediction",
}

func RunConcurrent(cfg Config, workersList []int) (benchmark.Report, error) {
	cfg = cfg.normalize()
	if err := validateWorkers(workersList); err != nil {
		return benchmark.Report{}, err
	}

	results := make([]benchmark.WorkerResult, 0, len(workersList))
	bestWorkers := workersList[0]
	bestTime := 0.0

	var last runOutput
	for _, workers := range workersList {
		fmt.Fprintf(os.Stderr, "[concurrent] workers=%d started\n", workers)
		times := make([]float64, 0, cfg.Runs)
		stages := make([]benchmark.StageTimes, 0, cfg.Runs)

		for run := 0; run < cfg.Runs; run++ {
			fmt.Fprintf(os.Stderr, "[concurrent] workers=%d run %d/%d started\n", workers, run+1, cfg.Runs)
			out, err := runOnce(cfg, workers, true)
			if err != nil {
				return benchmark.Report{}, err
			}
			fmt.Fprintf(os.Stderr, "[concurrent] workers=%d run %d/%d finished in %.3fs\n", workers, run+1, cfg.Runs, out.StageTimes.Total)
			last = out
			times = append(times, out.StageTimes.Total)
			stages = append(stages, out.StageTimes)
		}

		trimmed := benchmark.TrimmedMean(times)
		if len(results) == 0 || trimmed < bestTime {
			bestTime = trimmed
			bestWorkers = workers
		}

		results = append(results, benchmark.WorkerResult{
			Workers:            workers,
			TimesSeconds:       times,
			AvgSeconds:         benchmark.Average(times),
			TrimmedMeanSeconds: trimmed,
			Metrics:            last.Metrics,
			StageTimes:         benchmark.AverageStageTimes(stages),
			SplitMethod:        last.SplitMethod,
			ConcurrentSections: append([]string(nil), concurrentSections...),
		})
	}

	return benchmark.Report{
		Mode:                     "concurrent",
		DatasetRows:              last.DatasetRows,
		TrainRows:                last.TrainRows,
		TestRows:                 last.TestRows,
		Runs:                     cfg.Runs,
		Config:                   cfg.benchmarkConfig(),
		ResultsByWorkers:         results,
		BestWorkersByTrimmedMean: bestWorkers,
	}, nil
}

func RunCompare(cfg Config, workersList []int) (benchmark.CompareReport, error) {
	seq, err := RunSequential(cfg)
	if err != nil {
		return benchmark.CompareReport{}, err
	}

	conc, err := RunConcurrent(cfg, workersList)
	if err != nil {
		return benchmark.CompareReport{}, err
	}

	return benchmark.CompareReport{
		Mode:       "compare",
		Sequential: seq,
		Concurrent: conc,
	}, nil
}
