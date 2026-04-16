package csvio

import (
	"encoding/csv"
	"fmt"
	"os"
)

type Writer struct {
	file *os.File
	csv  *csv.Writer
}

func NewWriter(path string) (*Writer, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("create output CSV: %w", err)
	}

	w := csv.NewWriter(f)
	return &Writer{file: f, csv: w}, nil
}

func (w *Writer) WriteHeader(header []string) error {
	if err := w.csv.Write(header); err != nil {
		return fmt.Errorf("write CSV header: %w", err)
	}
	return nil
}

func (w *Writer) WriteRecord(record []string) error {
	if err := w.csv.Write(record); err != nil {
		return fmt.Errorf("write CSV record: %w", err)
	}
	return nil
}

func (w *Writer) Close() error {
	w.csv.Flush()
	if err := w.csv.Error(); err != nil {
		_ = w.file.Close()
		return fmt.Errorf("flush output CSV: %w", err)
	}
	if err := w.file.Close(); err != nil {
		return fmt.Errorf("close output CSV: %w", err)
	}
	return nil
}
