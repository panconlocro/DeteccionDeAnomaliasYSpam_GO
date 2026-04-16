package csvio

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"

	"timestamp-creation/internal/model"
)

func OpenCSVReader(path string) (*os.File, *csv.Reader, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open input CSV: %w", err)
	}
	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	r.ReuseRecord = false
	return f, r, nil
}

func ReadHeader(r *csv.Reader) ([]string, error) {
	record, err := r.Read()
	if err != nil {
		if err == io.EOF {
			return nil, fmt.Errorf("input CSV is empty")
		}
		return nil, fmt.Errorf("read CSV header: %w", err)
	}
	return append([]string(nil), record...), nil
}

func StreamRows(ctx context.Context, r *csv.Reader, startIndex int, out chan<- model.Row) error {
	idx := startIndex
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		record, err := r.Read()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("read CSV row %d: %w", idx, err)
		}

		row := model.Row{Index: idx, Record: append([]string(nil), record...)}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case out <- row:
		}
		idx++
	}
}
