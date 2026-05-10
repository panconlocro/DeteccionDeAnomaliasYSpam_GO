

package writer

import (
	"encoding/csv"
	"os"
	"sort"
	"strconv"
	"strings"

	"detecciondeanomalias/code/generateData/internal/model"
)

type CSVWriter struct {

	Writer *csv.Writer

	OriginalColumns []string
}

func NewCSVWriter(
	path string,
	originalColumns []string,
) (*CSVWriter, error) {

	f, err := os.Create(path)

	if err != nil {
		return nil, err
	}

	w := csv.NewWriter(f)

	sort.Strings(originalColumns)

	header := append(
		originalColumns,

		"TIMESTAMP",
		"DETALLE_QUEJA",
		"ES_SPAM",
		"SPAM_TAGS",
		"SPAM_SCORE",
	)

	err = w.Write(header)

	if err != nil {
		return nil, err
	}

	return &CSVWriter{
		Writer:          w,
		OriginalColumns: originalColumns,
	}, nil
}

func (w *CSVWriter) Write(
	c model.Complaint,
) error {

	row := []string{}

	for _, col := range w.OriginalColumns {

		row = append(
			row,
			c.OriginalData[col],
		)
	}

	row = append(
		row,

		c.Timestamp.Format(
			"2006-01-02 15:04:05",
		),

		c.DetalleQueja,

		boolToString(c.EsSpam),

		strings.Join(
			c.SpamTags,
			";",
		),

		strconv.FormatFloat(
			c.SpamScore,
			'f',
			2,
			64,
		),
	)

	return w.Writer.Write(row)
}

func boolToString(v bool) string {

	if v {
		return "1"
	}

	return "0"
}