// package main

// import (
//     "encoding/csv"
//     "log"
//     "math/rand"
//     "os"
//     "time"

//     "detecciondeanomalias/code/generateData/internal/anomly"
//     "detecciondeanomalias/code/generateData/internal/generator"
//     "detecciondeanomalias/code/generateData/internal/model"
//     "detecciondeanomalias/code/generateData/internal/sampler"
//     "detecciondeanomalias/code/generateData/internal/textgen"
//     "detecciondeanomalias/code/generateData/internal/timeline"
//     "detecciondeanomalias/code/generateData/internal/validator"
//     "detecciondeanomalias/code/generateData/internal/writer"
// )

// func loadCSV(path string) ([]sampler.SourceRow, error) {

// 	file, err := os.Open(path)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer file.Close()

// 	r := csv.NewReader(file)

// 	header, err := r.Read()
// 	if err != nil {
// 		return nil, err
// 	}

// 	var rows []sampler.SourceRow

// 	for {

// 		record, err := r.Read()
// 		if err != nil {
// 			break
// 		}

// 		row := sampler.SourceRow{}

// 		for i, value := range record {
// 			if i < len(header) {
// 				row[header[i]] = value
// 			}
// 		}

// 		rows = append(rows, row)
// 	}

// 	return rows, nil
// }

// func extractColumns(
// 	row sampler.SourceRow,
// ) []string {

// 	cols := []string{}

// 	for k := range row {
// 		cols = append(cols, k)
// 	}

// 	return cols
// }

// func main() {

// 	// =========================================================
// 	// CONFIG
// 	// =========================================================

// 	inputCSV := "data/clean/expedientes_clean.csv"

// 	outputCSV := "data/synthetic/expedientes_synthetic.csv"

// 	totalRows := 1_000_000

// 	seed := time.Now().UnixNano()

// 	// =========================================================
// 	// RNG
// 	// =========================================================

// 	rng := rand.New(rand.NewSource(seed))

// 	// =========================================================
// 	// LOAD SOURCE DATA
// 	// =========================================================

// 	log.Println("Loading source dataset...")

// 	rows, err := loadCSV(inputCSV)
// 	if err != nil {
// 		log.Fatalf("cannot load csv: %v", err)
// 	}

// 	if len(rows) == 0 {
// 		log.Fatal("dataset is empty")
// 	}

// 	log.Printf("Loaded %d base rows\n", len(rows))

// 	// =========================================================
// 	// CONTEXT
// 	// =========================================================

// 	ctx := model.NewContext()

// 	// =========================================================
// 	// COMPONENTS
// 	// =========================================================

// 	s := sampler.RandomSampler{
// 		Rows: rows,
// 		RNG:  rng,
// 	}

// 	baseGen := generator.BaseGenerator{
// 		RNG: rng,
// 	}

// 	textGen := textgen.Generator{
// 		RNG: rng,
// 	}

// 	timelineEngine := timeline.Engine{
// 		RNG: rng,
// 	}

// 	anomalyEngine := anomaly.Engine{
// 		RNG: rng,
// 		Available: []anomaly.Anomaly{

// 			&anomaly.BurstAnomaly{},

// 			&anomaly.NightAnomaly{},

// 			&anomaly.FakeEntityAnomaly{
// 				RNG: rng,
// 			},

// 			&anomaly.DuplicateAnomaly{
// 				Text: &textGen,
// 			},
// 		},
// 	}

// 	v := validator.Validator{}

// 	w, err := writer.NewCSVWriter(outputCSV)
// 	if err != nil {
// 		log.Fatalf("cannot create writer: %v", err)
// 	}

// 	// =========================================================
// 	// GENERATION LOOP
// 	// =========================================================

// 	log.Println("Starting synthetic generation...")

// 	start := time.Now()

// 	for i := 0; i < totalRows; i++ {

// 		// -----------------------------------------------------
// 		// SAMPLE REAL ROW
// 		// -----------------------------------------------------

// 		row := s.Sample()

// 		// -----------------------------------------------------
// 		// GENERATE BASE COMPLAINT
// 		// -----------------------------------------------------

// 		complaint := baseGen.Generate(row)

// 		// -----------------------------------------------------
// 		// GENERATE TIMESTAMP
// 		// -----------------------------------------------------

// 		baseDate := time.Now().AddDate(
// 			0,
// 			0,
// 			-rng.Intn(365),
// 		)

// 		complaint.FechaHora =
// 			timelineEngine.GenerateNormalTimestamp(baseDate)

// 		// -----------------------------------------------------
// 		// GENERATE NORMAL TEXT
// 		// -----------------------------------------------------

// 		complaint.Detalle =
// 			textGen.GenerateNormal(
// 				complaint.Denunciado,
// 				complaint.Materia,
// 			)

// 		// -----------------------------------------------------
// 		// APPLY ANOMALIES
// 		// -----------------------------------------------------

// 		anomalyEngine.ApplyRandom(
// 			&complaint,
// 			ctx,
// 		)

// 		// -----------------------------------------------------
// 		// VALIDATE
// 		// -----------------------------------------------------

// 		if !v.Validate(&complaint) {
// 			continue
// 		}

// 		// -----------------------------------------------------
// 		// UPDATE CONTEXT
// 		// -----------------------------------------------------

// 		ctx.RecentComplaints =
// 			append(ctx.RecentComplaints, complaint)

// 		ctx.EntityFrequency[
// 			complaint.Denunciado,
// 		]++

// 		// evitar crecimiento infinito
// 		if len(ctx.RecentComplaints) > 50000 {

// 			ctx.RecentComplaints =
// 				ctx.RecentComplaints[1000:]
// 		}

// 		// -----------------------------------------------------
// 		// WRITE OUTPUT
// 		// -----------------------------------------------------

// 		err := w.Write(complaint)
// 		if err != nil {
// 			log.Fatalf("write error: %v", err)
// 		}

// 		// -----------------------------------------------------
// 		// PROGRESS
// 		// -----------------------------------------------------

// 		if (i+1)%100000 == 0 {

// 			elapsed := time.Since(start)

// 			log.Printf(
// 				"%d/%d generated (%v)",
// 				i+1,
// 				totalRows,
// 				elapsed,
// 			)
// 		}
// 	}

// 	// =========================================================
// 	// FLUSH WRITER
// 	// =========================================================

// 	w.Writer.Flush()

// 	// =========================================================
// 	// FINAL
// 	// =========================================================

// 	duration := time.Since(start)

// 	log.Printf(
// 		"Finished. Generated %d rows in %v",
// 		totalRows,
// 		duration,
// 	)
// }

package main

import (
	"encoding/csv"
	"io"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	anomaly "detecciondeanomalias/code/generateData/internal/anomly"
	"detecciondeanomalias/code/generateData/internal/generator"
	"detecciondeanomalias/code/generateData/internal/model"
	"detecciondeanomalias/code/generateData/internal/sampler"
	"detecciondeanomalias/code/generateData/internal/textgen"
	"detecciondeanomalias/code/generateData/internal/timeline"
	"detecciondeanomalias/code/generateData/internal/validator"
	"detecciondeanomalias/code/generateData/internal/writer"
)

func loadCSV(
	path string,
) ([]sampler.SourceRow, error) {

	file, err := os.Open(path)

	if err != nil {
		return nil, err
	}

	defer file.Close()

	r := csv.NewReader(file)

	r.FieldsPerRecord = -1

	header, err := r.Read()

	if err != nil {
		return nil, err
	}

	var rows []sampler.SourceRow

	for {

		record, err := r.Read()

		if err != nil {

			if err == io.EOF {
				break
			}

			return nil, err
		}

		row := sampler.SourceRow{}

		for i, value := range record {

			if i < len(header) {

				row[header[i]] = value
			}
		}

		rows = append(rows, row)
	}

	return rows, nil
}

func extractColumns(
	row sampler.SourceRow,
) []string {

	cols := []string{}

	for k := range row {
		cols = append(cols, k)
	}

	return cols
}

func main() {

	// =====================================================
	// CONFIG
	// =====================================================

	inputCSV :=
		"data/clean/expedientes_clean.csv"

	outputCSV :=
		"data/synthetic/expedientes_synthetic.csv"

	outputDir := filepath.Dir(outputCSV)

	if outputDir != "." {

		err := os.MkdirAll(outputDir, 0o755)

		if err != nil {

			log.Fatalf(
				"cannot create output directory: %v",
				err,
			)
		}
	}

	totalRows := 1_000_000

	seed := time.Now().UnixNano()

	// =====================================================
	// RNG
	// =====================================================

	rng := rand.New(
		rand.NewSource(seed),
	)

	// =====================================================
	// LOAD DATASET
	// =====================================================

	log.Println(
		"Loading source dataset...",
	)

	rows, err := loadCSV(inputCSV)

	if err != nil {

		log.Fatalf(
			"cannot load csv: %v",
			err,
		)
	}

	if len(rows) == 0 {

		log.Fatal(
			"dataset is empty",
		)
	}

	log.Printf(
		"Loaded %d source rows",
		len(rows),
	)

	// =====================================================
	// EXTRACT ORIGINAL COLUMNS
	// =====================================================

	columns :=
		extractColumns(rows[0])

	// =====================================================
	// CONTEXT
	// =====================================================

	ctx := model.NewContext()

	// =====================================================
	// COMPONENTS
	// =====================================================

	s := sampler.RandomSampler{
		Rows: rows,
		RNG:  rng,
	}

	baseGen := generator.BaseGenerator{}

	textGen := textgen.Generator{
		RNG: rng,
	}

	timelineEngine := timeline.Engine{
		RNG: rng,
	}

	anomalyEngine := anomaly.Engine{
		RNG: rng,

		Available: []anomaly.Anomaly{

			&anomaly.BurstAnomaly{},

			&anomaly.NightAnomaly{},

			&anomaly.FakeEntityAnomaly{
				RNG: rng,
			},

			&anomaly.DuplicateAnomaly{
				Text: &textGen,
			},
		},
	}

	v := validator.Validator{}

	w, err := writer.NewCSVWriter(
		outputCSV,
		columns,
	)

	if err != nil {

		log.Fatalf(
			"cannot create writer: %v",
			err,
		)
	}

	// =====================================================
	// START
	// =====================================================

	log.Println(
		"Starting synthetic generation...",
	)

	start := time.Now()

	// =====================================================
	// MAIN LOOP
	// =====================================================

	for i := 0; i < totalRows; i++ {

		// -------------------------------------------------
		// SAMPLE REAL ROW
		// -------------------------------------------------

		row := s.Sample()

		// -------------------------------------------------
		// CREATE BASE COMPLAINT
		// -------------------------------------------------

		complaint :=
			baseGen.Generate(row)

		// -------------------------------------------------
		// BASE DATE
		// -------------------------------------------------

		fechaPres :=
			row["FECHA_PRESENTACION_pres"]

		baseDate, err := time.Parse(
			"2006-01-02",
			fechaPres,
		)

		if err != nil {

			baseDate =
				time.Now().AddDate(
					0,
					0,
					-rng.Intn(365),
				)
		}

		// -------------------------------------------------
		// SYNTHETIC TIMESTAMP
		// -------------------------------------------------

		complaint.Timestamp =
			timelineEngine.
				GenerateNormalTimestamp(
					baseDate,
				)

		// -------------------------------------------------
		// EXTRACT FEATURES
		// -------------------------------------------------

		denunciado :=
			row["DENUNCIADOS_pres"]

		if denunciado == "" {

			denunciado =
				"empresa"
		}

		materia :=
			row["MATERIA_pres"]

		if materia == "" {

			materia =
				"servicio"
		}

		// -------------------------------------------------
		// GENERATE SYNTHETIC TEXT
		// -------------------------------------------------

		complaint.DetalleQueja =
			textGen.GenerateNormal(
				denunciado,
				materia,
			)

		// -------------------------------------------------
		// APPLY ANOMALIES
		// -------------------------------------------------

		anomalyEngine.ApplyRandom(
			&complaint,
			ctx,
		)

		// -------------------------------------------------
		// SPAM SCORE
		// -------------------------------------------------

		if complaint.EsSpam {

			complaint.SpamScore =
				0.7 +
					rng.Float64()*0.3

		} else {

			complaint.SpamScore =
				rng.Float64() * 0.3
		}

		// -------------------------------------------------
		// UPDATE CONTEXT
		// -------------------------------------------------

		ctx.RecentComplaints =
			append(
				ctx.RecentComplaints,
				complaint,
			)

		den :=
			complaint.OriginalData["DENUNCIADOS_pres"]

		ctx.EntityFrequency[den]++

		// evitar crecimiento infinito
		if len(ctx.RecentComplaints) > 50000 {

			ctx.RecentComplaints =
				ctx.RecentComplaints[1000:]
		}

		// -------------------------------------------------
		// VALIDATE
		// -------------------------------------------------

		if !v.Validate(
			&complaint,
		) {
			continue
		}

		// -------------------------------------------------
		// WRITE CSV
		// -------------------------------------------------

		err = w.Write(
			complaint,
		)

		if err != nil {

			log.Fatalf(
				"write error: %v",
				err,
			)
		}

		// -------------------------------------------------
		// PROGRESS
		// -------------------------------------------------

		if (i+1)%100000 == 0 {

			elapsed :=
				time.Since(start)

			log.Printf(
				"%d/%d generated (%v)",
				i+1,
				totalRows,
				elapsed,
			)
		}
	}

	// =====================================================
	// FLUSH
	// =====================================================

	w.Writer.Flush()

	// =====================================================
	// FINAL
	// =====================================================

	duration := time.Since(start)

	log.Printf(
		"Finished generating %d rows in %v",
		totalRows,
		duration,
	)
}
