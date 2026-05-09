package text

import (
	"reflect"
	"testing"
)

func TestTokenizeNormalizesSpanishText(t *testing.T) {
	got := Tokenize("Queja por TELÉFONO!!  Número 123", false)
	want := []string{"queja", "por", "telefono", "numero", "123"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("tokens = %#v, want %#v", got, want)
	}
}

func TestTokenizeAddsBigrams(t *testing.T) {
	got := Tokenize("Queja banco", true)
	want := []string{"queja", "banco", "queja_banco"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("tokens = %#v, want %#v", got, want)
	}
}
