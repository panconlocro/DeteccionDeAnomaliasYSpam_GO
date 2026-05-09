package model

import "testing"

func TestMetricsFinalize(t *testing.T) {
	var m Metrics
	m.Add(1, 1)
	m.Add(1, 0)
	m.Add(0, 1)
	m.Add(0, 0)

	got := m.Finalize()
	if got.TP != 1 || got.TN != 1 || got.FP != 1 || got.FN != 1 {
		t.Fatalf("unexpected confusion matrix: %+v", got)
	}
	if got.Accuracy != 0.5 || got.Precision != 0.5 || got.Recall != 0.5 || got.F1 != 0.5 {
		t.Fatalf("unexpected metrics: %+v", got)
	}
}
