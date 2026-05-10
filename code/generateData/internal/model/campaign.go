package model

import "time"

type Campaign struct {
    StartTime time.Time
    EndTime   time.Time

    TargetEntity string

    MessageVariants []string

    BurstProbability float64
}