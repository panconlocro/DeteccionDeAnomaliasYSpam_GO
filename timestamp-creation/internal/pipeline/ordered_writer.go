package pipeline

import (
	"fmt"

	"timestamp-creation/internal/model"
)

// WriteFunc receives rows in stable order.
type WriteFunc func(row model.ProcessedRow) error

// OrderedWriter receives out-of-order rows and writes only when the next
// expected index is available.
type OrderedWriter struct {
	nextIndex int
	pending   map[int]model.ProcessedRow
	writeFn   WriteFunc
}

func NewOrderedWriter(startIndex int, writeFn WriteFunc) (*OrderedWriter, error) {
	if writeFn == nil {
		return nil, fmt.Errorf("write function cannot be nil")
	}
	if startIndex < 0 {
		return nil, fmt.Errorf("startIndex must be >= 0")
	}

	return &OrderedWriter{
		nextIndex: startIndex,
		pending:   make(map[int]model.ProcessedRow),
		writeFn:   writeFn,
	}, nil
}

func (o *OrderedWriter) Push(row model.ProcessedRow) error {
	if row.Index < o.nextIndex {
		return fmt.Errorf("received late/duplicate row index %d; next expected is %d", row.Index, o.nextIndex)
	}
	if _, exists := o.pending[row.Index]; exists {
		return fmt.Errorf("received duplicate row index %d", row.Index)
	}

	o.pending[row.Index] = row
	return o.flushContiguous()
}

// Finalize validates that no gaps remain.
func (o *OrderedWriter) Finalize() error {
	if len(o.pending) == 0 {
		return nil
	}
	return fmt.Errorf("pipeline finished with %d pending rows; next expected index %d", len(o.pending), o.nextIndex)
}

func (o *OrderedWriter) flushContiguous() error {
	for {
		row, ok := o.pending[o.nextIndex]
		if !ok {
			return nil
		}

		if err := o.writeFn(row); err != nil {
			return fmt.Errorf("write failed at row index %d: %w", row.Index, err)
		}

		delete(o.pending, o.nextIndex)
		o.nextIndex++
	}
}
