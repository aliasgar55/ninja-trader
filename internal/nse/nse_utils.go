package nse

import (
	"bufio"
	"bytes"
	"io"
	"time"
)

type DateRange [2]time.Time

func SplitDateRangeByYear(from, to time.Time) []DateRange {
	var ranges []DateRange
	if from.Equal(to) {
		ranges = append(ranges, DateRange{from, to})
	}
	for d := from; !d.After(to); d = d.AddDate(1, 0, 0) {
		toCurr := d.AddDate(1, 0, -1)
		if toCurr.After(to){
			toCurr = to
		}
		ranges = append(ranges, DateRange{d, toCurr})
	}
	return ranges
}

func skipBOM(r io.Reader) (io.Reader, error) {
	br := bufio.NewReader(r)

	b, err := br.Peek(3)
	if err == nil && bytes.Equal(b, []byte{0xEF, 0xBB, 0xBF}) {
		_, _ = br.Discard(3)
	}

	return br, nil
}
