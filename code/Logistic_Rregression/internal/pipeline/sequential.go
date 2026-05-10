package pipeline

import (
	"fmt"
	"os"

	"detecciondeanomalias/code/Logistic_Rregression/internal/benchmark"
)

func RunSequential(cfg Config) (benchmark.Report, error) {
	cfg = cfg.normalize()
	times := make([]float64, 0, cfg.Runs)
	totalTimes := make([]float64, 0, cfg.Runs)
	readTimes := make([]float64, 0, cfg.Runs)
	stages := make([]benchmark.StageTimes, 0, cfg.Runs)

	var last runOutput
	for run := 0; run < cfg.Runs; run++ {
		fmt.Fprintf(os.Stderr, "[sequential] run %d/%d started\n", run+1, cfg.Runs)
		out, err := runOnce(cfg, 1, false)
		if err != nil {
			return benchmark.Report{}, err
		}
		fmt.Fprintf(os.Stderr, "[sequential] run %d/%d finished in %.3fs\n", run+1, cfg.Runs, out.StageTimes.Total)
		last = out
		times = append(times, benchmark.ProcessingSeconds(out.StageTimes))
		totalTimes = append(totalTimes, out.StageTimes.Total)
		readTimes = append(readTimes, out.StageTimes.ReadCSV)
		stages = append(stages, out.StageTimes)
	}

	result := benchmark.WorkerResult{
		Workers:                 1,
		TimesSeconds:            times,
		TotalTimesSeconds:       totalTimes,
		ReadCSVSeconds:          readTimes,
		AvgSeconds:              benchmark.Average(times),
		TrimmedMeanSeconds:      benchmark.TrimmedMean(times),
		AvgTotalSeconds:         benchmark.Average(totalTimes),
		TrimmedMeanTotalSeconds: benchmark.TrimmedMean(totalTimes),
		Metrics:                 last.Metrics,
		StageTimes:              benchmark.AverageStageTimes(stages),
		SplitMethod:             last.SplitMethod,
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
