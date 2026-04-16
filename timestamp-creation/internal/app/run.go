package app

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"timestamp-creation/internal/clean"
	"timestamp-creation/internal/config"
	"timestamp-creation/internal/csvio"
	"timestamp-creation/internal/generator"
	"timestamp-creation/internal/model"
	"timestamp-creation/internal/pipeline"
	"timestamp-creation/internal/stats"
)

// Run executes the two-pass pipeline:
// pass 1 profiles repetition groups; pass 2 enriches rows concurrently.
func Run(ctx context.Context, cfg config.Config) (stats.Snapshot, error) {
	header, indexes, profiles, err := buildProfiles(ctx, cfg)
	if err != nil {
		return stats.Snapshot{}, err
	}

	collector := stats.NewCollector()
	if err := enrichAndWrite(ctx, cfg, header, indexes, profiles, collector); err != nil {
		return collector.Snapshot(), err
	}

	return collector.Snapshot(), nil
}

func buildProfiles(
	ctx context.Context,
	cfg config.Config,
) ([]string, model.ColumnIndexes, map[model.ProfileKey]model.GroupProfile, error) {
	inputFile, reader, err := csvio.OpenCSVReader(cfg.InputPath)
	if err != nil {
		return nil, model.ColumnIndexes{}, nil, err
	}
	defer inputFile.Close()

	header, err := csvio.ReadHeader(reader)
	if err != nil {
		return nil, model.ColumnIndexes{}, nil, err
	}

	indexes, err := csvio.DetectColumnIndexes(header)
	if err != nil {
		return nil, model.ColumnIndexes{}, nil, err
	}

	profiles := make(map[model.ProfileKey]model.GroupProfile)
	rows := make(chan model.Row, cfg.ChannelBuffer)
	readErrCh := make(chan error, 1)

	go func() {
		defer close(rows)
		readErrCh <- csvio.StreamRows(ctx, reader, 0, rows)
	}()

	for row := range rows {
		key := generator.BuildProfileKey(row.Record, indexes.Required)
		profile := profiles[key]
		profile.Count++
		if profile.States == nil {
			profile.States = make(map[string]int)
		}
		state := clean.NormalizeText(safeField(row.Record, indexes.Required.State))
		if state == "" {
			state = "unknown"
		}
		profile.States[state]++
		profiles[key] = profile
	}

	if err := <-readErrCh; err != nil {
		return nil, model.ColumnIndexes{}, nil, err
	}

	return header, indexes, profiles, nil
}

func enrichAndWrite(
	ctx context.Context,
	cfg config.Config,
	header []string,
	indexes model.ColumnIndexes,
	profiles map[model.ProfileKey]model.GroupProfile,
	collector *stats.Collector,
) error {
	gen, err := generator.New(cfg)
	if err != nil {
		return err
	}
	profileStore := generator.NewProfileStore(profiles)

	writer, err := csvio.NewWriter(cfg.OutputPath)
	if err != nil {
		return err
	}
	defer writer.Close()

	outHeader := csvio.BuildOutputHeader(header, cfg.AddPatternColumns)
	if err := writer.WriteHeader(outHeader); err != nil {
		return err
	}

	jobs := make(chan model.Row, cfg.ChannelBuffer)
	readerErrCh := make(chan error, 1)
	workerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		defer close(jobs)
		readerErrCh <- streamSecondPass(workerCtx, cfg, jobs, header)
	}()

	workerFn := func(_ context.Context, _ int, row model.Row) (model.ProcessedRow, error) {
		key := generator.BuildProfileKey(row.Record, indexes.Required)
		profile := profileStore.Get(key)

		rawDate := safeField(row.Record, indexes.Required.DateReceived)
		ts, decision, enrichErr := gen.EnrichDateReceived(row, rawDate, key, profile)

		record := csvio.AppendSyntheticColumns(
			row.Record,
			ts,
			cfg.AddPatternColumns,
			string(decision.Type),
			decision.CampaignID,
			boolString(decision.SeededSuspicious),
		)

		collector.Observe(decision, enrichErr)
		return model.ProcessedRow{
			Index:    row.Index,
			Record:   record,
			Decision: decision,
			ParseErr: enrichErr,
		}, nil
	}

	writeFn := func(row model.ProcessedRow) error {
		return writer.WriteRecord(row.Record)
	}

	pipeErr := pipeline.RunOrdered(workerCtx, pipeline.Options{
		Workers:      cfg.Workers,
		ResultBuffer: cfg.ChannelBuffer,
		StartIndex:   0,
	}, jobs, workerFn, writeFn)

	cancel()
	readErr := <-readerErrCh

	if err := errors.Join(pipeErr, readErr); err != nil {
		return err
	}

	return nil
}

func streamSecondPass(ctx context.Context, cfg config.Config, out chan<- model.Row, headerPass1 []string) error {
	inputFile, reader, err := csvio.OpenCSVReader(cfg.InputPath)
	if err != nil {
		return err
	}
	defer inputFile.Close()

	headerPass2, err := csvio.ReadHeader(reader)
	if err != nil {
		return err
	}
	if !headersCompatible(headerPass1, headerPass2) {
		return fmt.Errorf("input header changed between pass 1 and pass 2")
	}

	return csvio.StreamRows(ctx, reader, 0, out)
}

func headersCompatible(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if strings.TrimSpace(a[i]) != strings.TrimSpace(b[i]) {
			return false
		}
	}
	return true
}

func safeField(record []string, idx int) string {
	if idx < 0 || idx >= len(record) {
		return ""
	}
	return record[idx]
}

func boolString(v bool) string {
	if v {
		return "true"
	}
	return "false"
}
