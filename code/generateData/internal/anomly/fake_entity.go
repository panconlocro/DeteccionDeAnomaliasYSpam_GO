// package anomaly

// import (
//     "fmt"
//     "math/rand"

//     "detecciondeanomalias/code/generateData/internal/model"
// )

// type FakeEntityAnomaly struct {
//     RNG *rand.Rand
// }

// var prefixes = []string{
//     "Corporación",
//     "Grupo",
//     "Servicios",
//     "Consorcio",
// }

// var cores = []string{
//     "Integral",
//     "Andino",
//     "Financiero",
//     "Comercial",
// }

// var suffixes = []string{
//     "SAC",
//     "EIRL",
//     "SA",
// }

// func (a *FakeEntityAnomaly) Name() string {
//     return "fake_entity"
// }

// func (a *FakeEntityAnomaly) Apply(c *model.Complaint, ctx *model.GenerationContext) {

//     c.EsSpam = true

//     c.SpamTags = append(c.SpamTags, a.Name())

//     p := prefixes[a.RNG.Intn(len(prefixes))]
//     m := cores[a.RNG.Intn(len(cores))]
//     s := suffixes[a.RNG.Intn(len(suffixes))]

//     c.Denunciado = fmt.Sprintf("%s %s %s", p, m, s)
// }

package anomaly

import (
	"fmt"
	"math/rand"

	"detecciondeanomalias/code/generateData/internal/model"
)

type FakeEntityAnomaly struct {
	RNG *rand.Rand
}

var prefixes = []string{
	"Corporación",
	"Grupo",
	"Servicios",
	"Consorcio",
}

var cores = []string{
	"Integral",
	"Andino",
	"Financiero",
	"Comercial",
}

var suffixes = []string{
	"SAC",
	"EIRL",
	"SA",
}

func (a *FakeEntityAnomaly) Name() string {
	return "fake_entity"
}

func (a *FakeEntityAnomaly) Apply(
	c *model.Complaint,
	ctx *model.GenerationContext,
) {

	c.EsSpam = true

	c.SpamTags = append(
		c.SpamTags,
		a.Name(),
	)

	p := prefixes[
		a.RNG.Intn(len(prefixes))]

	m := cores[
		a.RNG.Intn(len(cores))]
	

	s := suffixes[
		a.RNG.Intn(len(suffixes))]
	

	c.OriginalData["DENUNCIADOS_pres"] =
		fmt.Sprintf(
			"%s %s %s",
			p,
			m,
			s,
		)
}