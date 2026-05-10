

package anomaly

import (
	"time"

	"detecciondeanomalias/code/generateData/internal/model"
)

type NightAnomaly struct{}

func (a *NightAnomaly) Name() string {
	return "night_activity"
}

func (a *NightAnomaly) Apply(
	c *model.Complaint,
	ctx *model.GenerationContext,
) {

	c.EsSpam = true

	c.SpamTags = append(
		c.SpamTags,
		a.Name(),
	)

	c.Timestamp = time.Date(
		c.Timestamp.Year(),
		c.Timestamp.Month(),
		c.Timestamp.Day(),
		2,
		10,
		0,
		0,
		time.UTC,
	)
}