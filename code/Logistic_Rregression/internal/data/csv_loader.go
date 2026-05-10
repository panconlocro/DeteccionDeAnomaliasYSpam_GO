package data

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"detecciondeanomalias/code/Logistic_Rregression/internal/features"
)

type Record struct {
	Index        int
	Text         string
	Label        int
	Timestamp    time.Time
	HasTimestamp bool
	Materia      string
	Tipo         string
	Tokens       []string
}

func LoadRecords(path string, limit int) ([]Record, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	header, err := reader.Read()
	if err != nil {
		return nil, err
	}

	idx := buildIndex(header)
	textIdx, ok := idx["DETALLE_QUEJA"]
	if !ok {
		return nil, fmt.Errorf("columna requerida no encontrada: DETALLE_QUEJA")
	}

	labelIdx, ok := idx["ES_SPAM"]
	if !ok {
		return nil, fmt.Errorf("columna requerida no encontrada: ES_SPAM")
	}

	timestampIdx := optionalIndex(idx, "TIMESTAMP")
	dateIdx := firstOptionalIndex(idx, "FECHA_PRESENTACION_pres", "FECHA_PRESENTACION")
	hourIdx := optionalIndex(idx, "HORA_PRESENTACION")
	materiaIdx := firstOptionalIndex(idx, "MATERIA", "MATERIA_pres")
	tipoIdx := firstOptionalIndex(idx, "TIPO_EXPEDIENTE", "TIPO_EXPEDIENTE_pres")

	records := make([]Record, 0)
	rowNumber := 1
	for {
		row, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("error leyendo CSV en fila %d: %w", rowNumber+1, err)
		}
		rowNumber++

		label, err := parseLabel(get(row, labelIdx))
		if err != nil {
			return nil, fmt.Errorf("ES_SPAM invalido en fila %d: %w", rowNumber, err)
		}

		timestamp, hasTimestamp := features.ParseTimestamp(
			get(row, timestampIdx),
			get(row, dateIdx),
			get(row, hourIdx),
		)

		records = append(records, Record{
			Index:        len(records),
			Text:         get(row, textIdx),
			Label:        label,
			Timestamp:    timestamp,
			HasTimestamp: hasTimestamp,
			Materia:      get(row, materiaIdx),
			Tipo:         get(row, tipoIdx),
		})

		if limit > 0 && len(records) >= limit {
			break
		}
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("dataset vacio")
	}

	return records, nil
}

func buildIndex(header []string) map[string]int {
	idx := make(map[string]int, len(header))
	for i, col := range header {
		idx[strings.TrimSpace(col)] = i
	}
	return idx
}

func optionalIndex(idx map[string]int, name string) int {
	if i, ok := idx[name]; ok {
		return i
	}
	return -1
}

func firstOptionalIndex(idx map[string]int, names ...string) int {
	for _, name := range names {
		if i, ok := idx[name]; ok {
			return i
		}
	}
	return -1
}

func get(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

func parseLabel(value string) (int, error) {
	value = strings.TrimSpace(strings.ToLower(value))
	switch value {
	case "1", "true", "t", "spam", "si", "sí", "yes":
		return 1, nil
	case "0", "false", "f", "no", "":
		return 0, nil
	}

	if f, err := strconv.ParseFloat(value, 64); err == nil {
		if f >= 0.5 {
			return 1, nil
		}
		return 0, nil
	}

	return 0, fmt.Errorf("valor %q no convertible a etiqueta binaria", value)
}
