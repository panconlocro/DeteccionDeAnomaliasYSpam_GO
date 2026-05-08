package timeline

import (
    "math/rand"
    "time"
)

type Engine struct {
    RNG *rand.Rand
}

func (e *Engine) GenerateNormalTimestamp(base time.Time) time.Time {

    hourWeights := []int{
        1, 1, 1, 1, 1,
        2, 3, 5, 12, 18,
        20, 18, 15, 14, 13,
        12, 10, 8, 5, 4,
        3, 2, 1, 1,
    }

    total := 0
    for _, w := range hourWeights {
        total += w
    }

    pick := e.RNG.Intn(total)

    acc := 0
    hour := 9

    for h, w := range hourWeights {
        acc += w
        if pick < acc {
            hour = h
            break
        }
    }

    minute := e.RNG.Intn(60)
    second := e.RNG.Intn(60)

    return time.Date(
        base.Year(),
        base.Month(),
        base.Day(),
        hour,
        minute,
        second,
        0,
        time.UTC,
    )
}