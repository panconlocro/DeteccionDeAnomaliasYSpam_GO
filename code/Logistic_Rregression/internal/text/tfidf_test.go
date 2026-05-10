package text

import "testing"

func TestTFIDFSmallCorpus(t *testing.T) {
	docs := [][]string{
		{"queja", "banco", "queja"},
		{"queja", "internet"},
	}

	vectorizer := BuildVectorizer(docs, 10, 1)
	if vectorizer.FeatureCount() != 3 {
		t.Fatalf("feature count = %d, want 3", vectorizer.FeatureCount())
	}

	vec := vectorizer.Vectorize([]string{"queja", "banco"})
	if len(vec) != 2 {
		t.Fatalf("vector length = %d, want 2", len(vec))
	}

	if _, ok := vectorizer.Vocab["queja"]; !ok {
		t.Fatal("expected token queja in vocabulary")
	}
}
