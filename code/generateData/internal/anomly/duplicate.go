// package anomaly

// import (
//     "detecciondeanomalias/code/generateData/internal/model"
//     "detecciondeanomalias/code/generateData/internal/textgen"
// )

// type DuplicateAnomaly struct {
//     Text *textgen.Generator
// }

// func (a *DuplicateAnomaly) Name() string {
//     return "duplicate_text"
// }

// func (a *DuplicateAnomaly) Apply(c *model.Complaint, ctx *model.GenerationContext) {

//     c.EsSpam = true

//     c.SpamTags = append(c.SpamTags, a.Name())

//     c.Detalle = a.Text.GenerateDuplicateVariant(c.Detalle)
// }

package anomaly

import (
	"detecciondeanomalias/code/generateData/internal/model"
	"detecciondeanomalias/code/generateData/internal/textgen"
)

type DuplicateAnomaly struct {
	Text *textgen.Generator
}

func (a *DuplicateAnomaly) Name() string {
	return "duplicate_text"
}

func (a *DuplicateAnomaly) Apply(
	c *model.Complaint,
	ctx *model.GenerationContext,
) {

	c.EsSpam = true

	c.SpamTags = append(
		c.SpamTags,
		a.Name(),
	)

	c.DetalleQueja =
		a.Text.GenerateDuplicateVariant(
			c.DetalleQueja,
		)
}