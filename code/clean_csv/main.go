package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultInput  = "data/complaints-2026-04-14_21_03.enriched.csv"
	defaultOutput = "data/complaints-2026-04-14_21_03.cleaned.csv"
	defaultQC     = "data/complaints-2026-04-14_21_03.qc.json"
)

var (
	redactionRegex       = regexp.MustCompile(`(?i)X{2,}`)
	yearPlaceholderRegex = regexp.MustCompile(`(?i)XX/XX/year>`)
	nullTokens           = map[string]struct{}{
		"nan":  {},
		"na":   {},
		"n/a":  {},
		"null": {},
		"none": {},
	}
	dateLayouts = []string{
		"01/02/06",
		"01/02/2006",
		"2006-01-02",
	}
)

type QCReport struct {
	GeneratedAt            string         `json:"generated_at"`
	Source                 string         `json:"source"`
	Output                 string         `json:"output"`
	RowsIn                 int            `json:"rows_in"`
	RowsOut                int            `json:"rows_out"`
	RowsDeduped            int            `json:"rows_deduped"`
	RowsShort              int            `json:"rows_short"`
	RowsLong               int            `json:"rows_long"`
	Missingness            map[string]int `json:"missingness"`
	ParseErrors            map[string]int `json:"parse_errors"`
	SyntheticPatternCounts map[string]int `json:"synthetic_pattern_counts,omitempty"`
	NarrativeLength        LengthStats    `json:"narrative_length"`
	UniqueComplaintIDs     int            `json:"unique_complaint_ids"`
}

type LengthStats struct {
	Count int     `json:"count"`
	Min   int     `json:"min"`
	Max   int     `json:"max"`
	Avg   float64 `json:"avg"`
}

type lengthAccumulator struct {
	count int
	min   int
	max   int
	sum   int64
}

type rowJob struct {
	seq      int
	row      []string
	wasShort bool
	wasLong  bool
}

type rowResult struct {
	seq                  int
	row                  []string
	wasShort             bool
	wasLong              bool
	parseErrDateReceived bool
	parseErrDateSent     bool
	parseErrSyntheticTS  bool
}

func (a *lengthAccumulator) Add(n int) {
	if n <= 0 {
		return
	}
	if a.count == 0 || n < a.min {
		a.min = n
	}
	if a.count == 0 || n > a.max {
		a.max = n
	}
	a.count++
	a.sum += int64(n)
}

func (a lengthAccumulator) Stats() LengthStats {
	avg := 0.0
	if a.count > 0 {
		avg = float64(a.sum) / float64(a.count)
	}
	return LengthStats{
		Count: a.count,
		Min:   a.min,
		Max:   a.max,
		Avg:   avg,
	}
}

func main() {
	inPath := flag.String("in", defaultInput, "Input CSV path")
	outPath := flag.String("out", defaultOutput, "Output CSV path")
	qcPath := flag.String("qc", defaultQC, "QC report JSON path")
	dedup := flag.Bool("dedup", true, "Deduplicate rows")
	limit := flag.Int("limit", 0, "Max rows to process (0 = all)")
	workers := flag.Int("workers", runtime.NumCPU(), "Number of worker goroutines")
	flag.Parse()

	if err := run(*inPath, *outPath, *qcPath, *dedup, *limit, *workers); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(inPath, outPath, qcPath string, dedup bool, limit int, workers int) error {
	if filepath.Clean(inPath) == filepath.Clean(outPath) {
		return errors.New("input and output paths must differ")
	}

	inFile, err := os.Open(inPath)
	if err != nil {
		return fmt.Errorf("open input: %w", err)
	}
	defer inFile.Close()

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	outFile, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("create output: %w", err)
	}
	defer outFile.Close()

	reader := csv.NewReader(bufio.NewReader(inFile))
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true

	header, err := reader.Read()
	if err == io.EOF {
		return errors.New("input is empty")
	}
	if err != nil {
		return fmt.Errorf("read header: %w", err)
	}

	if len(header) == 0 {
		return errors.New("input has no columns")
	}

	header[0] = strings.TrimPrefix(header[0], "\ufeff")
	colIdx := make(map[string]int, len(header))
	for i, name := range header {
		trimmed := strings.TrimSpace(name)
		header[i] = trimmed
		colIdx[strings.ToLower(trimmed)] = i
	}

	idxDateReceived := findIndex(colIdx, "date received")
	idxDateSent := findIndex(colIdx, "date sent to company")
	idxNarrative := findIndex(colIdx, "consumer complaint narrative")
	idxComplaintID := findIndex(colIdx, "complaint id")
	idxZip := findIndex(colIdx, "zip code")
	idxSyntheticTS := findIndex(colIdx, "synthetic_date_received_ts")
	idxPattern := findIndex(colIdx, "synthetic_pattern_type")
	idxCompany := findIndex(colIdx, "company")
	idxProduct := findIndex(colIdx, "product")
	idxIssue := findIndex(colIdx, "issue")

	missingness := make(map[string]int, len(header))
	for _, name := range header {
		missingness[name] = 0
	}

	parseErrors := make(map[string]int)
	patternCounts := make(map[string]int)
	var narrativeStats lengthAccumulator

	var seenDedup map[string]struct{}
	if dedup {
		seenDedup = make(map[string]struct{})
	}
	seenComplaintIDs := make(map[string]struct{})

	bufferedOut := bufio.NewWriter(outFile)
	writer := csv.NewWriter(bufferedOut)
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	workerCount := workers
	if workerCount < 1 {
		workerCount = 1
	}

	rowsIn := 0
	rowsOut := 0
	rowsDeduped := 0
	rowsShort := 0
	rowsLong := 0

	fallbackIdx := []int{idxDateReceived, idxCompany, idxProduct, idxIssue, idxNarrative}

	jobs := make(chan rowJob, workerCount*2)
	results := make(chan rowResult, workerCount*2)
	errCh := make(chan error, 1)

	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				results <- cleanRow(job, idxComplaintID, idxZip, idxDateReceived, idxDateSent, idxSyntheticTS, idxNarrative)
			}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	go func() {
		seq := 0
		for {
			if limit > 0 && seq >= limit {
				break
			}
			row, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				sendError(errCh, fmt.Errorf("read row %d: %w", seq+1, err))
				break
			}

			wasShort := false
			wasLong := false
			if len(row) < len(header) {
				wasShort = true
				row = append(row, make([]string, len(header)-len(row))...)
			} else if len(row) > len(header) {
				wasLong = true
				row = row[:len(header)]
			}

			jobs <- rowJob{
				seq:      seq,
				row:      row,
				wasShort: wasShort,
				wasLong:  wasLong,
			}
			seq++
		}
		close(jobs)
	}()

	pending := make(map[int]rowResult)
	nextSeq := 0

	for result := range results {
		pending[result.seq] = result
		for {
			current, ok := pending[nextSeq]
			if !ok {
				break
			}
			delete(pending, nextSeq)

			rowsIn++
			if current.wasShort {
				rowsShort++
			}
			if current.wasLong {
				rowsLong++
			}

			if current.parseErrDateReceived {
				parseErrors["Date received"]++
			}
			if current.parseErrDateSent {
				parseErrors["Date sent to company"]++
			}
			if current.parseErrSyntheticTS {
				parseErrors["synthetic_date_received_ts"]++
			}

			row := current.row

			if idxNarrative >= 0 && idxNarrative < len(row) {
				if row[idxNarrative] != "" {
					narrativeStats.Add(len(row[idxNarrative]))
				}
			}

			if idxPattern >= 0 && idxPattern < len(row) {
				if row[idxPattern] != "" {
					patternCounts[row[idxPattern]]++
				}
			}

			if idxComplaintID >= 0 && idxComplaintID < len(row) {
				if row[idxComplaintID] != "" {
					seenComplaintIDs[row[idxComplaintID]] = struct{}{}
				}
			}

			for i := range row {
				if row[i] == "" {
					missingness[header[i]]++
				}
			}

			shouldWrite := true
			if dedup {
				key := dedupKey(row, idxComplaintID, fallbackIdx)
				if _, ok := seenDedup[key]; ok {
					rowsDeduped++
					shouldWrite = false
				} else {
					seenDedup[key] = struct{}{}
				}
			}

			if shouldWrite {
				if err := writer.Write(row); err != nil {
					return fmt.Errorf("write row %d: %w", rowsIn, err)
				}
				rowsOut++
			}
			nextSeq++
		}
	}

	if len(pending) > 0 {
		return errors.New("missing rows while ordering output")
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("flush output: %w", err)
	}
	if err := bufferedOut.Flush(); err != nil {
		return fmt.Errorf("flush buffer: %w", err)
	}

	if err := firstError(errCh); err != nil {
		return err
	}

	report := QCReport{
		GeneratedAt:            time.Now().UTC().Format(time.RFC3339),
		Source:                 inPath,
		Output:                 outPath,
		RowsIn:                 rowsIn,
		RowsOut:                rowsOut,
		RowsDeduped:            rowsDeduped,
		RowsShort:              rowsShort,
		RowsLong:               rowsLong,
		Missingness:            missingness,
		ParseErrors:            parseErrors,
		SyntheticPatternCounts: patternCounts,
		NarrativeLength:        narrativeStats.Stats(),
		UniqueComplaintIDs:     len(seenComplaintIDs),
	}

	if err := os.MkdirAll(filepath.Dir(qcPath), 0o755); err != nil {
		return fmt.Errorf("create qc dir: %w", err)
	}

	qcFile, err := os.Create(qcPath)
	if err != nil {
		return fmt.Errorf("create qc report: %w", err)
	}
	defer qcFile.Close()

	encoder := json.NewEncoder(qcFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		return fmt.Errorf("write qc report: %w", err)
	}

	return nil
}

func cleanRow(job rowJob, idxComplaintID, idxZip, idxDateReceived, idxDateSent, idxSyntheticTS, idxNarrative int) rowResult {
	row := job.row
	for i := range row {
		row[i] = cleanValue(row[i])
	}

	if idxComplaintID >= 0 && idxComplaintID < len(row) {
		row[idxComplaintID] = normalizeNumericID(row[idxComplaintID])
	}
	if idxZip >= 0 && idxZip < len(row) {
		row[idxZip] = normalizeNumericID(row[idxZip])
	}

	parseErrDateReceived := false
	if idxDateReceived >= 0 && idxDateReceived < len(row) {
		normalized, ok := normalizeDate(row[idxDateReceived])
		if row[idxDateReceived] != "" && !ok {
			parseErrDateReceived = true
		}
		row[idxDateReceived] = normalized
	}

	parseErrDateSent := false
	if idxDateSent >= 0 && idxDateSent < len(row) {
		normalized, ok := normalizeDate(row[idxDateSent])
		if row[idxDateSent] != "" && !ok {
			parseErrDateSent = true
		}
		row[idxDateSent] = normalized
	}

	parseErrSyntheticTS := false
	if idxSyntheticTS >= 0 && idxSyntheticTS < len(row) {
		normalized, ok := normalizeTimestamp(row[idxSyntheticTS])
		if row[idxSyntheticTS] != "" && !ok {
			parseErrSyntheticTS = true
		}
		row[idxSyntheticTS] = normalized
	}

	if idxNarrative >= 0 && idxNarrative < len(row) {
		if row[idxNarrative] != "" {
			row[idxNarrative] = normalizeNarrative(row[idxNarrative])
		}
	}

	return rowResult{
		seq:                  job.seq,
		row:                  row,
		wasShort:             job.wasShort,
		wasLong:              job.wasLong,
		parseErrDateReceived: parseErrDateReceived,
		parseErrDateSent:     parseErrDateSent,
		parseErrSyntheticTS:  parseErrSyntheticTS,
	}
}

func sendError(errCh chan<- error, err error) {
	select {
	case errCh <- err:
	default:
	}
}

func firstError(errCh <-chan error) error {
	select {
	case err := <-errCh:
		return err
	default:
		return nil
	}
}

func findIndex(idx map[string]int, name string) int {
	key := strings.ToLower(strings.TrimSpace(name))
	if value, ok := idx[key]; ok {
		return value
	}
	return -1
}

func cleanValue(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if isNullToken(trimmed) {
		return ""
	}
	return trimmed
}

func isNullToken(value string) bool {
	token := strings.ToLower(strings.TrimSpace(value))
	_, ok := nullTokens[token]
	return ok
}

func normalizeNumericID(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	dot := strings.IndexByte(trimmed, '.')
	if dot == -1 {
		return trimmed
	}
	if !isAllDigits(trimmed[:dot]) {
		return trimmed
	}
	if !isAllZeros(trimmed[dot+1:]) {
		return trimmed
	}
	return trimmed[:dot]
}

func isAllDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func isAllZeros(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r != '0' {
			return false
		}
	}
	return true
}

func normalizeDate(value string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", true
	}
	for _, layout := range dateLayouts {
		if parsed, err := time.Parse(layout, trimmed); err == nil {
			return parsed.Format("2006-01-02"), true
		}
	}
	return trimmed, false
}

func normalizeTimestamp(value string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", true
	}
	if parsed, err := time.Parse(time.RFC3339, trimmed); err == nil {
		return parsed.UTC().Format(time.RFC3339), true
	}
	if parsed, err := time.Parse(time.RFC3339Nano, trimmed); err == nil {
		return parsed.UTC().Format(time.RFC3339Nano), true
	}
	return trimmed, false
}

func normalizeNarrative(value string) string {
	cleaned := strings.ReplaceAll(value, "\r\n", "\n")
	cleaned = strings.ReplaceAll(cleaned, "\r", "\n")
	cleaned = strings.ReplaceAll(cleaned, "\n", " ")
	cleaned = strings.TrimSpace(cleaned)
	if cleaned == "" {
		return ""
	}
	cleaned = normalizeRedactions(cleaned)
	cleaned = collapseSpaces(cleaned)
	return cleaned
}

func normalizeRedactions(value string) string {
	value = yearPlaceholderRegex.ReplaceAllString(value, "XX/XX/YYYY")
	value = redactionRegex.ReplaceAllString(value, "XXXX")
	return value
}

func collapseSpaces(value string) string {
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return ""
	}
	return strings.Join(fields, " ")
}

func dedupKey(row []string, idxComplaintID int, fallbackIdx []int) string {
	if idxComplaintID >= 0 && idxComplaintID < len(row) {
		if row[idxComplaintID] != "" {
			return "id:" + row[idxComplaintID]
		}
	}

	hasher := fnv.New64a()
	for _, idx := range fallbackIdx {
		if idx < 0 || idx >= len(row) {
			continue
		}
		value := row[idx]
		if len(value) > 2048 {
			value = value[:2048]
		}
		_, _ = hasher.Write([]byte(value))
		_, _ = hasher.Write([]byte{0})
	}

	return "hash:" + strconv.FormatUint(hasher.Sum64(), 16)
}
