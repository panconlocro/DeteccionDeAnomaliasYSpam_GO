package model

import "time"

type Complaint struct {

	// Todas las columnas originales
	OriginalData map[string]string

	// Timestamp sintético derivado
	Timestamp time.Time

	// Texto generado
	DetalleQueja string

	// Labels
	EsSpam bool

	SpamTags []string

	SpamScore float64

	Metadata map[string]any
}