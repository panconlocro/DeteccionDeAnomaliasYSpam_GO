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

func AverageResourceUsage(values []ResourceUsage) ResourceUsage {
	if len(values) == 0 {
		return ResourceUsage{}
	}

	var out ResourceUsage
	for _, value := range values {
		out.WallSeconds += value.WallSeconds
		out.CPUPercent += value.CPUPercent
		out.CPUUserSeconds += value.CPUUserSeconds
		out.CPUSystemSeconds += value.CPUSystemSeconds
		out.CPUTotalSeconds += value.CPUTotalSeconds
		out.MaxRSSMB += value.MaxRSSMB
		out.MaxHeapAllocMB += value.MaxHeapAllocMB
		out.MaxHeapSysMB += value.MaxHeapSysMB
		out.MaxGoSysMB += value.MaxGoSysMB
		out.TotalAllocDeltaMB += value.TotalAllocDeltaMB
		out.NumGCDelta += value.NumGCDelta
	}

	scale := 1 / float64(len(values))
	out.WallSeconds *= scale
	out.CPUPercent *= scale
	out.CPUUserSeconds *= scale
	out.CPUSystemSeconds *= scale
	out.CPUTotalSeconds *= scale
	out.MaxRSSMB *= scale
	out.MaxHeapAllocMB *= scale
	out.MaxHeapSysMB *= scale
	out.MaxGoSysMB *= scale
	out.TotalAllocDeltaMB *= scale
	out.NumGCDelta *= scale

	return out
}
