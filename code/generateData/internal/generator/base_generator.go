// package generator

// import (
//     "fmt"
//     "math/rand"
//     "strings"
//     "time"

//     "detecciondeanomalias/code/generateData/internal/model"
//     "detecciondeanomalias/code/generateData/internal/sampler"
// )

// type BaseGenerator struct {
//     RNG *rand.Rand
// }

// func (g *BaseGenerator) Generate(row sampler.SourceRow) model.Complaint {
//     now := time.Now()

//     return model.Complaint{
//         ID: fmt.Sprintf("CMP-%d", g.RNG.Int63()),

//         FechaHora: now,

//         Materia: normalize(row["MATERIA_pres"], "servicio"),
//         TipoExpediente: normalize(row["TIPO_EXPEDIENTE_pres"], "queja"),
//         Denunciado: normalize(row["DENUNCIADOS_pres"], "empresa"),

//         Detalle: "",

//         Metadata: map[string]any{},
//     }
// }

// func normalize(v string, fallback string) string {
//     v = strings.TrimSpace(v)
//     if v == "" {
//         return fallback
//     }
//     return v
// }

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