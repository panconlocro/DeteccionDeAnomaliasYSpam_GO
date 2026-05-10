
package anomaly

import (
	"time"

	"detecciondeanomalias/code/generateData/internal/model"
)

type BurstAnomaly struct{}

func (a *BurstAnomaly) Name() string {
	return "temporal_burst"
}

func (a *BurstAnomaly) Apply(
	c *model.Complaint,
	ctx *model.GenerationContext,
) {

	c.EsSpam = true

	c.SpamTags = append(
		c.SpamTags,
		a.Name(),
	)

	offset :=
		time.Duration(
			c.Timestamp.Second()%20,
		) * time.Second

	c.Timestamp =
		c.Timestamp.Add(offset)
}