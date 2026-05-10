package benchmark

import "sort"

func Average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, value := range values {
		sum += value
	}
	return sum / float64(len(values))
}

func TrimmedMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	if len(values) <= 2 {
		return Average(values)
	}

	copyValues := append([]float64(nil), values...)
	sort.Float64s(copyValues)
	return Average(copyValues[1 : len(copyValues)-1])
}

func AverageStageTimes(values []StageTimes) StageTimes {
	if len(values) == 0 {
		return StageTimes{}
	}

	var out StageTimes
	for _, value := range values {
		out.ReadCSV += value.ReadCSV
		out.Preprocess += value.Preprocess
		out.Vocabulary += value.Vocabulary
		out.Vectorization += value.Vectorization
		out.Training += value.Training
		out.Evaluation += value.Evaluation
		out.Total += value.Total
	}

	scale := 1 / float64(len(values))
	out.ReadCSV *= scale
	out.Preprocess *= scale
	out.Vocabulary *= scale
	out.Vectorization *= scale
	out.Training *= scale
	out.Evaluation *= scale
	out.Total *= scale

	return out
}

func ProcessingSeconds(stage StageTimes) float64 {
	seconds := stage.Total - stage.ReadCSV
	if seconds < 0 {
		return 0
	}
	return seconds
}
