package anomaly

import "detecciondeanomalias/code/generateData/internal/model"

type Anomaly interface {
    Apply(*model.Complaint, *model.GenerationContext)
    Name() string
}