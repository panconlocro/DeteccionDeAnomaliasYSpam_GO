

package generator

import (
	"strings"

	"detecciondeanomalias/code/generateData/internal/model"
	"detecciondeanomalias/code/generateData/internal/sampler"
)

type BaseGenerator struct{}

func (g *BaseGenerator) Generate(
	row sampler.SourceRow,
) model.Complaint {

	data := make(map[string]string)

	for k, v := range row {

		data[k] =
			strings.TrimSpace(v)
	}

	return model.Complaint{
		OriginalData: data,
		DetalleQueja: "",
		EsSpam: false,
		SpamTags: []string{},
		Metadata: map[string]any{},
	}
}