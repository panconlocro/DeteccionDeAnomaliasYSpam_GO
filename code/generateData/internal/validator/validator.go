

package validator

import (
	"detecciondeanomalias/code/generateData/internal/model"
)

type Validator struct{}

func (v *Validator) Validate(
	c *model.Complaint,
) bool {

	if c.DetalleQueja == "" {
		return false
	}

	denunciado :=
		c.OriginalData["DENUNCIADOS_pres"]

	if denunciado == "" {
		return false
	}

	return true
}