package benchmark

import "detecciondeanomalias/code/Logistic_Rregression/internal/model"

type ModelConfig struct {
	Epochs       int     `json:"epochs"`
	LearningRate float64 `json:"learning_rate"`
	LambdaL2     float64 `json:"lambda_l2"`
	BatchSize    int     `json:"batch_size"`
	MaxFeatures  int     `json:"max_features"`
	MinDF        int     `json:"min_df"`
	Threshold    float64 `json:"threshold"`
	Seed         int64   `json:"seed"`
	UseBigrams   bool    `json:"use_bigrams"`
	TestRatio    float64 `json:"test_ratio"`
}

type StageTimes struct {
	ReadCSV       float64 `json:"read_csv"`
	Preprocess    float64 `json:"preprocess"`
	Vocabulary    float64 `json:"vocabulary"`
	Vectorization float64 `json:"vectorization"`
	Training      float64 `json:"training"`
	Evaluation    float64 `json:"evaluation"`
	Total         float64 `json:"total"`
}

type WorkerResult struct {
	Workers                 int             `json:"workers"`
	TimesSeconds            []float64       `json:"times_seconds"`
	TotalTimesSeconds       []float64       `json:"total_times_seconds"`
	ReadCSVSeconds          []float64       `json:"read_csv_seconds"`
	AvgSeconds              float64         `json:"avg_seconds"`
	TrimmedMeanSeconds      float64         `json:"trimmed_mean_seconds"`
	AvgTotalSeconds         float64         `json:"avg_total_seconds"`
	TrimmedMeanTotalSeconds float64         `json:"trimmed_mean_total_seconds"`
	ResourceRuns            []ResourceUsage `json:"resource_runs"`
	ResourceUsage           ResourceUsage   `json:"resource_usage"`
	Metrics                 model.Metrics   `json:"metrics"`
	StageTimes              StageTimes      `json:"stage_times"`
	SplitMethod             string          `json:"split_method"`
	ConcurrentSections      []string        `json:"concurrent_sections,omitempty"`
}

type Report struct {
	Mode                     string         `json:"mode"`
	DatasetRows              int            `json:"dataset_rows"`
	TrainRows                int            `json:"train_rows"`
	TestRows                 int            `json:"test_rows"`
	Runs                     int            `json:"runs"`
	Config                   ModelConfig    `json:"config"`
	ResultsByWorkers         []WorkerResult `json:"results_by_workers"`
	BestWorkersByTrimmedMean int            `json:"best_workers_by_trimmed_mean,omitempty"`
}

type CompareReport struct {
	Mode       string `json:"mode"`
	Sequential Report `json:"sequential"`
	Concurrent Report `json:"concurrent"`
}
