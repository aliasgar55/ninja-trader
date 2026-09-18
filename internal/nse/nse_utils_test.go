package nse

import (
	"testing"
	"time"
)

func d(year, month, day int) time.Time {
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
}

func TestSplitDateRangeByYear_SingleDayRange(t *testing.T) {
	ranges := SplitDateRangeByYear(d(2024, 3, 15), d(2024, 3, 15))
	if len(ranges) != 1 {
		t.Fatalf("expected 1 range, got %d", len(ranges))
	}
	if ranges[0][0] != d(2024, 3, 15) || ranges[0][1] != d(2024, 3, 15) {
		t.Errorf("expected [2024-03-15, 2024-03-15], got [%v, %v]", ranges[0][0], ranges[0][1])
	}
}

func TestSplitDateRangeByYear_WithinOneYear(t *testing.T) {
	ranges := SplitDateRangeByYear(d(2024, 1, 1), d(2024, 6, 30))
	if len(ranges) != 1 {
		t.Fatalf("expected 1 range, got %d", len(ranges))
	}
	if ranges[0][0] != d(2024, 1, 1) || ranges[0][1] != d(2024, 6, 30) {
		t.Errorf("expected [2024-01-01, 2024-06-30], got [%v, %v]", ranges[0][0], ranges[0][1])
	}
}

func TestSplitDateRangeByYear_ExactlyOneYear(t *testing.T) {
	// from+1year == to, so the loop enters twice; second iteration clamps to `to`
	ranges := SplitDateRangeByYear(d(2024, 1, 1), d(2025, 1, 1))
	if len(ranges) != 2 {
		t.Fatalf("expected 2 ranges, got %d", len(ranges))
	}
	if ranges[0][0] != d(2024, 1, 1) || ranges[0][1] != d(2024, 12, 31) {
		t.Errorf("range 0: expected [2024-01-01, 2024-31-12], got [%v, %v]", ranges[0][0], ranges[0][1])
	}
	if ranges[1][0] != d(2025, 1, 1) || ranges[1][1] != d(2025, 1, 1) {
		t.Errorf("range 0: expected [2025-01-01, 2025-01-01], got [%v, %v]", ranges[0][0], ranges[0][1])
	}
}

func TestSplitDateRangeByYear_SpansMultipleYears(t *testing.T) {
	ranges := SplitDateRangeByYear(d(2020, 3, 1), d(2023, 6, 15))
	if len(ranges) != 4 {
		t.Fatalf("expected 4 ranges, got %d", len(ranges))
	}
	expected := []DateRange{
		{d(2020, 3, 1), d(2021, 2, 28)},
		{d(2021, 3, 1), d(2022, 2, 28)},
		{d(2022, 3, 1), d(2023, 2, 28)},
		{d(2023, 3, 1), d(2023, 6, 15)},
	}
	for i, r := range ranges {
		if r[0] != expected[i][0] || r[1] != expected[i][1] {
			t.Errorf("range %d: expected [%v, %v], got [%v, %v]", i, expected[i][0], expected[i][1], r[0], r[1])
		}
	}
}

func TestSplitDateRangeByYear_JustOverOneYear(t *testing.T) {
	ranges := SplitDateRangeByYear(d(2024, 1, 1), d(2025, 1, 2))
	if len(ranges) != 2 {
		t.Fatalf("expected 2 ranges, got %d", len(ranges))
	}
	if ranges[0][0] != d(2024, 1, 1) || ranges[0][1] != d(2024, 12, 31) {
		t.Errorf("range 0: expected [2024-01-01, 2025-01-01], got [%v, %v]", ranges[0][0], ranges[0][1])
	}
	if ranges[1][0] != d(2025, 1, 1) || ranges[1][1] != d(2025, 1, 2) {
		t.Errorf("range 1: expected [2025-01-01, 2025-01-02], got [%v, %v]", ranges[1][0], ranges[1][1])
	}
}


