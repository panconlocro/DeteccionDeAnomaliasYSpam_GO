package generator

import (
	"math"
	"math/rand"
	"time"

	"timestamp-creation/internal/model"
)

type clockRange struct {
	startSec int
	endSec   int
	weight   float64
}

var weekdayNormalRanges = []clockRange{
	{startSec: 0, endSec: 5*3600 - 1, weight: 0.03},
	{startSec: 5 * 3600, endSec: 8*3600 - 1, weight: 0.10},
	{startSec: 8 * 3600, endSec: 12*3600 - 1, weight: 0.32},
	{startSec: 12 * 3600, endSec: 17*3600 - 1, weight: 0.36},
	{startSec: 17 * 3600, endSec: 21*3600 - 1, weight: 0.15},
	{startSec: 21 * 3600, endSec: 86399, weight: 0.04},
}

var weekendNormalRanges = []clockRange{
	{startSec: 0, endSec: 6*3600 - 1, weight: 0.06},
	{startSec: 6 * 3600, endSec: 10*3600 - 1, weight: 0.14},
	{startSec: 10 * 3600, endSec: 16*3600 - 1, weight: 0.40},
	{startSec: 16 * 3600, endSec: 21*3600 - 1, weight: 0.28},
	{startSec: 21 * 3600, endSec: 86399, weight: 0.12},
}

var offHoursRanges = []clockRange{
	{startSec: 1 * 3600, endSec: 4*3600 + 59*60 + 59, weight: 0.86},
	{startSec: 0, endSec: 59*60 + 59, weight: 0.08},
	{startSec: 5 * 3600, endSec: 5*3600 + 59*60 + 59, weight: 0.06},
}

func pickPattern(
	r *rand.Rand,
	date time.Time,
	profile model.GroupProfile,
	key model.ProfileKey,
	cfg PatternConfig,
) model.PatternDecision {
	burst := cfg.BurstRatio
	off := cfg.OffHoursRatio
	coord := cfg.CoordinatedRatio
	regular := cfg.RegularIntervalsRatio

	multiplier := suspiciousMultiplier(profile.Count)
	if multiplier > 1 {
		burst *= multiplier
		off *= multiplier
		coord *= multiplier
		regular *= multiplier
	}

	totalSuspicious := burst + off + coord + regular
	normal := math.Max(0, cfg.NormalRatio)
	total := normal + totalSuspicious
	if total <= 0 {
		return model.PatternDecision{Type: model.PatternNormal}
	}

	x := r.Float64() * total
	if x < normal {
		return model.PatternDecision{Type: model.PatternNormal}
	}
	x -= normal

	if x < burst {
		return model.PatternDecision{Type: model.PatternBurst, CampaignID: BuildCampaignID(key), SeededSuspicious: true}
	}
	x -= burst

	if x < off {
		return model.PatternDecision{Type: model.PatternOffHours, CampaignID: BuildCampaignID(key), SeededSuspicious: true}
	}
	x -= off

	if x < coord {
		_ = date
		return model.PatternDecision{Type: model.PatternCoordinated, CampaignID: BuildCampaignID(key), SeededSuspicious: true}
	}

	return model.PatternDecision{Type: model.PatternRegularIntervals, CampaignID: BuildCampaignID(key), SeededSuspicious: true}
}

func secondsForPattern(
	r *rand.Rand,
	day time.Time,
	decision model.PatternDecision,
	rowIndex int,
	cfg PatternConfig,
) int {
	switch decision.Type {
	case model.PatternBurst:
		return secondsBurst(r, cfg.BurstWindowMinutes)
	case model.PatternOffHours:
		return sampleFromRanges(r, offHoursRanges)
	case model.PatternRegularIntervals:
		return secondsRegularIntervals(r, rowIndex)
	case model.PatternCoordinated:
		return secondsCoordinated(r, day)
	default:
		return secondsNormal(r, day)
	}
}

func secondsNormal(r *rand.Rand, day time.Time) int {
	wd := day.Weekday()
	if wd == time.Saturday || wd == time.Sunday {
		return sampleFromRanges(r, weekendNormalRanges)
	}
	return sampleFromRanges(r, weekdayNormalRanges)
}

func secondsBurst(r *rand.Rand, maxWindowMinutes int) int {
	if maxWindowMinutes < 1 {
		maxWindowMinutes = 1
	}
	window := r.Intn(maxWindowMinutes) + 1
	anchorMinute := r.Intn(24 * 60)
	offset := r.Intn(window*60 + 1)
	sec := anchorMinute*60 + offset
	if sec > 86399 {
		return 86399
	}
	return sec
}

func secondsRegularIntervals(r *rand.Rand, rowIndex int) int {
	base := 8*3600 + (rowIndex%120)*60
	jitter := r.Intn(41) - 20
	sec := base + jitter
	if sec < 0 {
		return 0
	}
	if sec > 86399 {
		return 86399
	}
	return sec
}

func secondsCoordinated(r *rand.Rand, day time.Time) int {
	campaignSeed := int(day.YearDay()%6) * 3600
	center := 9*3600 + campaignSeed
	jitter := r.Intn(25*60) - 12*60
	sec := center + jitter
	if sec < 0 {
		return 0
	}
	if sec > 86399 {
		return 86399
	}
	return sec
}

func sampleFromRanges(r *rand.Rand, ranges []clockRange) int {
	total := 0.0
	for _, rg := range ranges {
		total += rg.weight
	}
	if total <= 0 {
		return r.Intn(86400)
	}

	x := r.Float64() * total
	for _, rg := range ranges {
		if x < rg.weight {
			span := rg.endSec - rg.startSec + 1
			if span <= 0 {
				return rg.startSec
			}
			return rg.startSec + r.Intn(span)
		}
		x -= rg.weight
	}

	last := ranges[len(ranges)-1]
	span := last.endSec - last.startSec + 1
	if span <= 0 {
		return last.startSec
	}
	return last.startSec + r.Intn(span)
}

func suspiciousMultiplier(groupCount int) float64 {
	switch {
	case groupCount >= 50:
		return 2.2
	case groupCount >= 20:
		return 1.7
	case groupCount >= 10:
		return 1.3
	default:
		return 1.0
	}
}
