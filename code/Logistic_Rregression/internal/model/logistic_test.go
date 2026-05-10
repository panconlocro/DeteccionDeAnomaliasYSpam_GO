package model

import (
	"testing"

	"detecciondeanomalias/code/Logistic_Rregression/internal/features"
)

func TestSigmoid(t *testing.T) {
	if got := Sigmoid(0); got != 0.5 {
		t.Fatalf("Sigmoid(0) = %v, want 0.5", got)
	}
}

func TestLogisticPrediction(t *testing.T) {
	m := &LogisticRegression{
		Weights:   []float64{2},
		Bias:      -1,
		Threshold: 0.5,
	}

	pred := m.Predict(features.SparseVector{{Index: 0, Value: 1}})
	if pred != 1 {
		t.Fatalf("prediction = %d, want 1", pred)
	}
}

func TestTrainLearnsSimpleBoundary(t *testing.T) {
	examples := []Example{
		{X: features.SparseVector{{Index: 0, Value: 0}}, Y: 0},
		{X: features.SparseVector{{Index: 0, Value: 1}}, Y: 1},
	}

	m := Train(examples, 1, TrainConfig{
		Epochs:       200,
		LearningRate: 0.5,
		BatchSize:    1,
		Threshold:    0.5,
	})

	if m.Predict(examples[0].X) != 0 || m.Predict(examples[1].X) != 1 {
		t.Fatalf("model did not learn simple boundary")
	}
}
