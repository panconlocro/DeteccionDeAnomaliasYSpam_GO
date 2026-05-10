package anomaly

import (
    "math/rand"

    "detecciondeanomalias/code/generateData/internal/model"
)

type Engine struct {
    RNG *rand.Rand

    Available []Anomaly
}

func (e *Engine) ApplyRandom(c *model.Complaint, ctx *model.GenerationContext) {

    if e.RNG.Float64() < 0.82 {
        return
    }

    n := 1 + e.RNG.Intn(3)

    for i := 0; i < n; i++ {

        a := e.Available[
            e.RNG.Intn(len(e.Available))]

        a.Apply(c, ctx)
    }
}