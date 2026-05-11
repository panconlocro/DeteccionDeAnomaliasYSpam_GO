package pipeline

import (
	"fmt"
	"runtime"
	"sync"
	"time"

	"detecciondeanomalias/code/Logistic_Rregression/internal/benchmark"
	"detecciondeanomalias/code/Logistic_Rregression/internal/data"
	"detecciondeanomalias/code/Logistic_Rregression/internal/features"
	clfmodel "detecciondeanomalias/code/Logistic_Rregression/internal/model"
	txt "detecciondeanomalias/code/Logistic_Rregression/internal/text"
)

const maxCategoryLevels = 100

type runOutput struct {
	DatasetRows int
	TrainRows   int
	TestRows    int
	Metrics     clfmodel.Metrics
	StageTimes  benchmark.StageTimes
	Resources   benchmark.ResourceUsage
	SplitMethod string
}

func runOnce(cfg Config, workers int, concurrent bool) (runOutput, error) {
	cfg = cfg.normalize()
	if workers <= 0 {
		workers = runtime.NumCPU()
	}

	stopResourceMonitor := benchmark.StartResourceMonitor(25 * time.Millisecond)
	totalStart := time.Now()
	var stages benchmark.StageTimes

	start := time.Now()
	records, err := data.LoadRecords(cfg.Input, cfg.Limit)
	if err != nil {
		_ = stopResourceMonitor()
		return runOutput{}, err
	}
	stages.ReadCSV = time.Since(start).Seconds()

	start = time.Now()
	if concurrent {
		records = preprocessConcurrent(records, cfg.UseBigrams, workers)
	} else {
		records = preprocessSequential(records, cfg.UseBigrams)
	}
	stages.Preprocess = time.Since(start).Seconds()

	trainRecords, testRecords, splitMethod, err := data.Split(records, cfg.TestRatio, cfg.Seed)
	if err != nil {
		_ = stopResourceMonitor()
		return runOutput{}, err
	}

	start = time.Now()
	var vectorizer *txt.Vectorizer
	var catEncoder features.CategoryEncoder
	if concurrent {
		vectorizer = buildVectorizerConcurrent(trainRecords, cfg.MaxFeatures, cfg.MinDF, workers)
		catEncoder = buildCategoryEncoderConcurrent(trainRecords, workers)
	} else {
		vectorizer = buildVectorizerSequential(trainRecords, cfg.MaxFeatures, cfg.MinDF)
		catEncoder = buildCategoryEncoderSequential(trainRecords)
	}
	stages.Vocabulary = time.Since(start).Seconds()

	start = time.Now()
	dimension := vectorizer.FeatureCount() + features.TimestampFeatureCount + catEncoder.FeatureCount()
	var trainExamples []clfmodel.Example
	var testExamples []clfmodel.Example
	if concurrent {
		trainExamples = vectorizeConcurrent(trainRecords, vectorizer, catEncoder, dimension, workers)
		testExamples = vectorizeConcurrent(testRecords, vectorizer, catEncoder, dimension, workers)
	} else {
		trainExamples = vectorizeSequential(trainRecords, vectorizer, catEncoder, dimension)
		testExamples = vectorizeSequential(testRecords, vectorizer, catEncoder, dimension)
	}
	stages.Vectorization = time.Since(start).Seconds()

	trainCfg := clfmodel.TrainConfig{
		Epochs:       cfg.Epochs,
		LearningRate: cfg.LearningRate,
		LambdaL2:     cfg.LambdaL2,
		BatchSize:    cfg.BatchSize,
		Threshold:    cfg.Threshold,
	}

	start = time.Now()
	var model *clfmodel.LogisticRegression
	if concurrent {
		model = clfmodel.TrainConcurrent(trainExamples, dimension, trainCfg, workers)
	} else {
		model = clfmodel.Train(trainExamples, dimension, trainCfg)
	}
	stages.Training = time.Since(start).Seconds()

	start = time.Now()
	var metrics clfmodel.Metrics
	if concurrent {
		metrics = clfmodel.EvaluateConcurrent(model, testExamples, workers)
	} else {
		metrics = clfmodel.Evaluate(model, testExamples)
	}
	stages.Evaluation = time.Since(start).Seconds()
	stages.Total = time.Since(totalStart).Seconds()
	resources := stopResourceMonitor()

	return runOutput{
		DatasetRows: len(records),
		TrainRows:   len(trainRecords),
		TestRows:    len(testRecords),
		Metrics:     metrics,
		StageTimes:  stages,
		Resources:   resources,
		SplitMethod: splitMethod,
	}, nil
}

func preprocessSequential(records []data.Record, useBigrams bool) []data.Record {
	out := make([]data.Record, len(records))
	for i, record := range records {
		record.Tokens = txt.Tokenize(record.Text, useBigrams)
		out[i] = record
	}
	return out
}

func preprocessConcurrent(records []data.Record, useBigrams bool, workers int) []data.Record {
	out := make([]data.Record, len(records))
	jobs := make(chan int)

	var wg sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				record := records[i]
				record.Tokens = txt.Tokenize(record.Text, useBigrams)
				out[i] = record
			}
		}()
	}

	for i := range records {
		jobs <- i
	}
	close(jobs)
	wg.Wait()

	return out
}

func buildVectorizerSequential(records []data.Record, maxFeatures, minDF int) *txt.Vectorizer {
	docs := make([][]string, len(records))
	for i, record := range records {
		docs[i] = record.Tokens
	}
	return txt.BuildVectorizer(docs, maxFeatures, minDF)
}

func buildVectorizerConcurrent(records []data.Record, maxFeatures, minDF int, workers int) *txt.Vectorizer {
	actualWorkers := clampWorkers(workers, len(records))
	results := make(chan map[string]int, actualWorkers)
	chunks := chunkRanges(len(records), actualWorkers)

	var wg sync.WaitGroup
	for _, chunk := range chunks {
		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			results <- countDF(records[start:end])
		}(chunk.start, chunk.end)
	}

	wg.Wait()
	close(results)

	merged := make(map[string]int)
	for local := range results {
		for token, count := range local {
			merged[token] += count
		}
	}

	return txt.NewVectorizerFromDF(merged, len(records), maxFeatures, minDF)
}

func buildCategoryEncoderSequential(records []data.Record) features.CategoryEncoder {
	materias := make([]string, len(records))
	tipos := make([]string, len(records))
	for i, record := range records {
		materias[i] = record.Materia
		tipos[i] = record.Tipo
	}
	return features.BuildCategoryEncoder(materias, tipos, maxCategoryLevels)
}

func buildCategoryEncoderConcurrent(records []data.Record, workers int) features.CategoryEncoder {
	actualWorkers := clampWorkers(workers, len(records))
	type localCounts struct {
		materias map[string]int
		tipos    map[string]int
	}

	results := make(chan localCounts, actualWorkers)
	chunks := chunkRanges(len(records), actualWorkers)

	var wg sync.WaitGroup
	for _, chunk := range chunks {
		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			materias := make(map[string]int)
			tipos := make(map[string]int)
			for _, record := range records[start:end] {
				if value := features.NormalizeCategory(record.Materia); value != "" {
					materias[value]++
				}
				if value := features.NormalizeCategory(record.Tipo); value != "" {
					tipos[value]++
				}
			}
			results <- localCounts{materias: materias, tipos: tipos}
		}(chunk.start, chunk.end)
	}

	wg.Wait()
	close(results)

	materias := make(map[string]int)
	tipos := make(map[string]int)
	for local := range results {
		for value, count := range local.materias {
			materias[value] += count
		}
		for value, count := range local.tipos {
			tipos[value] += count
		}
	}

	return features.NewCategoryEncoderFromCounts(materias, tipos, maxCategoryLevels)
}

func countDF(records []data.Record) map[string]int {
	df := make(map[string]int)
	for _, record := range records {
		seen := make(map[string]struct{}, len(record.Tokens))
		for _, token := range record.Tokens {
			if token == "" {
				continue
			}
			seen[token] = struct{}{}
		}
		for token := range seen {
			df[token]++
		}
	}
	return df
}

func vectorizeSequential(
	records []data.Record,
	vectorizer *txt.Vectorizer,
	catEncoder features.CategoryEncoder,
	dimension int,
) []clfmodel.Example {
	out := make([]clfmodel.Example, len(records))
	for i, record := range records {
		out[i] = buildExample(record, vectorizer, catEncoder, dimension)
	}
	return out
}

func vectorizeConcurrent(
	records []data.Record,
	vectorizer *txt.Vectorizer,
	catEncoder features.CategoryEncoder,
	dimension int,
	workers int,
) []clfmodel.Example {
	out := make([]clfmodel.Example, len(records))
	jobs := make(chan int)

	var wg sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				out[i] = buildExample(records[i], vectorizer, catEncoder, dimension)
			}
		}()
	}

	for i := range records {
		jobs <- i
	}
	close(jobs)
	wg.Wait()

	return out
}

func buildExample(
	record data.Record,
	vectorizer *txt.Vectorizer,
	catEncoder features.CategoryEncoder,
	dimension int,
) clfmodel.Example {
	x := vectorizer.Vectorize(record.Tokens)

	timeBase := vectorizer.FeatureCount()
	timeFeatures := features.TimestampFeatures(record.Timestamp, record.HasTimestamp)
	for i, value := range timeFeatures {
		x = features.AppendNonZero(x, timeBase+i, value)
	}

	categoryBase := timeBase + features.TimestampFeatureCount
	x = catEncoder.Append(x, categoryBase, record.Materia, record.Tipo)

	return clfmodel.Example{
		X: x,
		Y: record.Label,
	}
}

type chunk struct {
	start int
	end   int
}

func chunkRanges(n int, workers int) []chunk {
	workers = clampWorkers(workers, n)
	chunkSize := (n + workers - 1) / workers
	chunks := make([]chunk, 0, workers)
	for start := 0; start < n; start += chunkSize {
		end := start + chunkSize
		if end > n {
			end = n
		}
		chunks = append(chunks, chunk{start: start, end: end})
	}
	return chunks
}

func clampWorkers(workers int, n int) int {
	if n <= 0 {
		return 1
	}
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	if workers > n {
		workers = n
	}
	return workers
}

func validateWorkers(workers []int) error {
	if len(workers) == 0 {
		return fmt.Errorf("debe indicar al menos un valor de workers")
	}
	for _, worker := range workers {
		if worker <= 0 {
			return fmt.Errorf("workers invalido: %d", worker)
		}
	}
	return nil
}
