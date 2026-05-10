package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"detecciondeanomalias/code/Logistic_Rregression/internal/benchmark"
	"detecciondeanomalias/code/Logistic_Rregression/internal/pipeline"
)

func main() {
	input := flag.String("input", "data/synthetic/expedientes_synthetic.csv", "CSV de entrada")
	mode := flag.String("mode", "sequential", "modo: sequential, concurrent o compare")
	runs := flag.Int("runs", 5, "numero de ejecuciones por configuracion")
	epochs := flag.Int("epochs", 10, "epocas de entrenamiento")
	lr := flag.Float64("lr", 0.05, "learning rate")
	lambda := flag.Float64("lambda", 0.001, "regularizacion L2")
	batchSize := flag.Int("batchSize", 1024, "tamano del mini-batch")
	maxFeatures := flag.Int("maxFeatures", 5000, "maximo de features TF-IDF")
	minDF := flag.Int("minDF", 2, "document frequency minima para vocabulario")
	threshold := flag.Float64("threshold", 0.5, "umbral de clasificacion")
	seed := flag.Int64("seed", 42, "seed para split aleatorio si no hay timestamp")
	testRatio := flag.Float64("testRatio", 0.2, "proporcion de test")
	workers := flag.String("workers", "1,2,4,8,16", "lista de workers separada por comas")
	out := flag.String("out", "", "ruta JSON de salida; si se omite, imprime en stdout")
	useBigrams := flag.Bool("bigrams", true, "incluir bigramas junto con unigramas")
	limit := flag.Int("limit", 0, "limite opcional de filas para pruebas rapidas")

	flag.Parse()

	cfg := pipeline.Config{
		Input:        *input,
		Runs:         *runs,
		Epochs:       *epochs,
		LearningRate: *lr,
		LambdaL2:     *lambda,
		BatchSize:    *batchSize,
		MaxFeatures:  *maxFeatures,
		MinDF:        *minDF,
		Threshold:    *threshold,
		Seed:         *seed,
		TestRatio:    *testRatio,
		UseBigrams:   *useBigrams,
		Limit:        *limit,
	}

	workerList, err := parseWorkers(*workers)
	if err != nil {
		exitErr(err)
	}

	var payload any
	switch normalizeMode(*mode) {
	case "sequential":
		payload, err = pipeline.RunSequential(cfg)
	case "concurrent":
		payload, err = pipeline.RunConcurrent(cfg, workerList)
	case "compare":
		payload, err = pipeline.RunCompare(cfg, workerList)
	default:
		err = fmt.Errorf("modo invalido %q; use sequential, concurrent o compare", *mode)
	}
	if err != nil {
		exitErr(err)
	}

	if err := writeJSON(*out, payload); err != nil {
		exitErr(err)
	}

	if strings.TrimSpace(*out) != "" {
		summaryPath := runSummaryPath(*out)
		if err := writeRunSummary(summaryPath, payload); err != nil {
			exitErr(err)
		}
		fmt.Fprintf(os.Stderr, "Resumen de ejecuciones escrito en %s\n", summaryPath)
	}
}

func parseWorkers(value string) ([]int, error) {
	parts := strings.Split(value, ",")
	workers := make([]int, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		n, err := strconv.Atoi(part)
		if err != nil || n <= 0 {
			return nil, fmt.Errorf("workers invalido: %q", part)
		}
		workers = append(workers, n)
	}
	if len(workers) == 0 {
		return nil, fmt.Errorf("debe indicar al menos un valor en -workers")
	}
	return workers, nil
}

func normalizeMode(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "secuencial":
		return "sequential"
	case "concurrente":
		return "concurrent"
	default:
		return value
	}
}

func writeJSON(path string, payload any) error {
	var file *os.File
	var err error

	if strings.TrimSpace(path) == "" {
		file = os.Stdout
	} else {
		dir := filepath.Dir(path)
		if dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return err
			}
		}
		file, err = os.Create(path)
		if err != nil {
			return err
		}
		defer file.Close()
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(payload)
}

func runSummaryPath(jsonPath string) string {
	ext := filepath.Ext(jsonPath)
	if ext == "" {
		return jsonPath + "_runs.txt"
	}
	return strings.TrimSuffix(jsonPath, ext) + "_runs.txt"
}

func writeRunSummary(path string, payload any) error {
	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	switch report := payload.(type) {
	case benchmark.Report:
		writeReportSummary(file, report)
	case benchmark.CompareReport:
		fmt.Fprintln(file, "REPORTE COMPARATIVO")
		fmt.Fprintln(file, strings.Repeat("=", 72))
		writeReportSummary(file, report.Sequential)
		fmt.Fprintln(file)
		writeReportSummary(file, report.Concurrent)
	default:
		return fmt.Errorf("tipo de reporte no soportado para resumen")
	}

	return nil
}

func writeReportSummary(file *os.File, report benchmark.Report) {
	fmt.Fprintf(file, "MODO: %s\n", strings.ToUpper(report.Mode))
	fmt.Fprintf(file, "Filas dataset: %d\n", report.DatasetRows)
	fmt.Fprintf(file, "Filas train: %d\n", report.TrainRows)
	fmt.Fprintf(file, "Filas test: %d\n", report.TestRows)
	fmt.Fprintf(file, "Runs: %d\n", report.Runs)
	if report.BestWorkersByTrimmedMean > 0 {
		fmt.Fprintf(file, "Mejor workers por media recortada sin lectura CSV: %d\n", report.BestWorkersByTrimmedMean)
	}
	fmt.Fprintln(file, strings.Repeat("-", 72))

	for _, result := range report.ResultsByWorkers {
		if report.Mode == "sequential" {
			fmt.Fprintln(file, "Configuracion: secuencial")
		} else {
			fmt.Fprintf(file, "Configuracion: workers=%d\n", result.Workers)
		}

		for i, seconds := range result.TimesSeconds {
			fmt.Fprintf(file, "Ejecucion %d: %.6f segundos (sin lectura CSV)\n", i+1, seconds)
		}

		alertsGenerated := result.Metrics.TP + result.Metrics.FP
		fmt.Fprintf(file, "Alertas generadas por el modelo (TP+FP): %d\n", alertsGenerated)
		fmt.Fprintf(file, "Alertas detectadas correctamente (TP): %d\n", result.Metrics.TP)
		fmt.Fprintf(file, "Alertas no detectadas (FN): %d\n", result.Metrics.FN)
		fmt.Fprintf(file, "Falsas alertas (FP): %d\n", result.Metrics.FP)
		fmt.Fprintf(file, "Tiempo lectura CSV promedio: %.6f segundos\n", result.StageTimes.ReadCSV)
		fmt.Fprintf(file, "Promedio de los stages sin lectura CSV: %.6f segundos\n", result.AvgSeconds)
		fmt.Fprintf(file, "Media recortada de los stages sin lectura CSV: %.6f segundos\n", result.TrimmedMeanSeconds)
		fmt.Fprintf(file, "Tiempo total promedio con lectura CSV: %.6f segundos\n", result.AvgTotalSeconds)
		fmt.Fprintf(file, "Media recortada total con lectura CSV: %.6f segundos\n", result.TrimmedMeanTotalSeconds)
		fmt.Fprintln(file, "Detalle promedio por stage:")
		fmt.Fprintf(file, "  read_csv: %.6f\n", result.StageTimes.ReadCSV)
		fmt.Fprintf(file, "  preprocess: %.6f\n", result.StageTimes.Preprocess)
		fmt.Fprintf(file, "  vocabulary: %.6f\n", result.StageTimes.Vocabulary)
		fmt.Fprintf(file, "  vectorization: %.6f\n", result.StageTimes.Vectorization)
		fmt.Fprintf(file, "  training: %.6f\n", result.StageTimes.Training)
		fmt.Fprintf(file, "  evaluation: %.6f\n", result.StageTimes.Evaluation)
		fmt.Fprintf(file, "  total: %.6f\n", result.StageTimes.Total)
		fmt.Fprintln(file, strings.Repeat("-", 72))
	}
}

func exitErr(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
