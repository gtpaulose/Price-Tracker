package tracker

import (
	"math"
	"time"
)

type (
	Record struct {
		Time         time.Time
		CurrencyPair string
		Rate
		Diff Diff
		Settings
	}

	Diff struct {
		Value      float64
		Percentage float64
	}
)

func NewRecord(cp string, rate Rate, diffValue, diffPerc float64, settings Settings) *Record {
	return &Record{time.Now(), cp, rate, newDiff(diffValue, diffPerc), settings}
}

// TODO: Dynamically change the rounded value based on settings.OscOscPercentage
// newDiff will create a new Diff object and limit the decimal places to 4
func newDiff(diffValue, diffPerc float64) Diff {
	return Diff{
		Value:      math.Round(diffValue*10000) / 10000,
		Percentage: math.Round(diffPerc*10000) / 10000,
	}
}

func calculateDiffPercentage(diff, original float64) float64 { return diff * 100 / original }
