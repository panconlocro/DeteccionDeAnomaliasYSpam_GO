package sampler

import (
    "math/rand"
)

type SourceRow map[string]string

type RandomSampler struct {
    Rows []SourceRow
    RNG  *rand.Rand
}

func (s *RandomSampler) Sample() SourceRow {
    return s.Rows[s.RNG.Intn(len(s.Rows))]
}