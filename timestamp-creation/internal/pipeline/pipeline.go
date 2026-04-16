package pipeline

import (
	"context"
	"errors"
	"fmt"

	"timestamp-creation/internal/model"
)

// Options configures the concurrent ordered pipeline.
type Options struct {
	Workers      int
	ResultBuffer int
	StartIndex   int
}

// RunOrdered executes a worker pool and guarantees write order by row index.
// It drains all worker results even after the first error to avoid goroutine leaks.
func RunOrdered(
	ctx context.Context,
	opts Options,
	in <-chan model.Row,
	workerFn WorkerFunc,
	writeFn WriteFunc,
) error {
	if err := opts.validate(); err != nil {
		return err
	}

	orderedWriter, err := NewOrderedWriter(opts.StartIndex, writeFn)
	if err != nil {
		return err
	}

	results, err := StartWorkerPool(ctx, opts.Workers, in, workerFn, opts.ResultBuffer)
	if err != nil {
		return err
	}

	var firstErr error
	for processed := range results {
		if firstErr != nil {
			continue
		}

		if err := orderedWriter.Push(processed); err != nil {
			firstErr = err
		}
	}

	finalizeErr := orderedWriter.Finalize()
	if finalizeErr != nil {
		firstErr = errors.Join(firstErr, finalizeErr)
	}

	return firstErr
}

func (o Options) validate() error {
	if o.Workers <= 0 {
		return fmt.Errorf("workers must be > 0")
	}
	if o.ResultBuffer < 0 {
		return fmt.Errorf("result buffer must be >= 0")
	}
	if o.StartIndex < 0 {
		return fmt.Errorf("start index must be >= 0")
	}
	return nil
}
