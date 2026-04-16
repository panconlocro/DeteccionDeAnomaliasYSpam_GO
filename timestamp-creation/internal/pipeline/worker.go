package pipeline

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"timestamp-creation/internal/model"
)

// WorkerFunc transforms one row into a processed row.
// The returned ProcessedRow must represent the same logical row.
type WorkerFunc func(ctx context.Context, workerID int, row model.Row) (model.ProcessedRow, error)

// StartWorkerPool launches a fixed-size worker pool that consumes rows from in
// and emits processed rows on a single output channel.
func StartWorkerPool(
	ctx context.Context,
	workers int,
	in <-chan model.Row,
	fn WorkerFunc,
	outBuffer int,
) (<-chan model.ProcessedRow, error) {
	if workers <= 0 {
		return nil, fmt.Errorf("workers must be > 0")
	}
	if in == nil {
		return nil, fmt.Errorf("input channel cannot be nil")
	}
	if fn == nil {
		return nil, fmt.Errorf("worker func cannot be nil")
	}
	if outBuffer < 0 {
		return nil, fmt.Errorf("outBuffer must be >= 0")
	}

	out := make(chan model.ProcessedRow, outBuffer)

	var wg sync.WaitGroup
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		workerID := i
		go func() {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				case row, ok := <-in:
					if !ok {
						return
					}

					processed, err := fn(ctx, workerID, row)
					if processed.Index != row.Index {
						// Ordered output depends on the original input index.
						processed.Index = row.Index
					}
					if processed.Record == nil {
						processed.Record = row.Record
					}
					if err != nil {
						processed.ParseErr = errors.Join(processed.ParseErr, err)
					}

					select {
					case <-ctx.Done():
						return
					case out <- processed:
					}
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out, nil
}
