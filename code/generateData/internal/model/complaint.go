package model

import "time"

type Complaint struct {

	// columnas originales
	OriginalData map[string]string

	// timestamp sintético
	Timestamp time.Time

	// texto sintético
	DetalleQueja string

	// labels
	EsSpam bool

	SpamTags []string

	SpamScore float64

	Metadata map[string]any
}