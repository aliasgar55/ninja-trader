package nse

import (
	"time"
)

type DateRange [2]time.Time

func SplitDateRangeByYear(from, to time.Time) []DateRange {
	var ranges []DateRange
	if from.Equal(to) {
		ranges = append(ranges, DateRange{from, to})
	}
	for d := from; !d.After(to) && !d.Equal(to); d = d.AddDate(1, 0, 0) {
		toCurr := d.AddDate(1, 0, 0)
		if toCurr.After(to){
			toCurr = to
		}
		ranges = append(ranges, DateRange{d, toCurr})
	}
	return ranges
}
