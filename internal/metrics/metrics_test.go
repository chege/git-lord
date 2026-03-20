package metrics

import (
	"testing"
	"time"
)

func TestCalculateHours(t *testing.T) {
	tests := []struct {
		name          string
		timestamps    []int64
		sessionWindow int
		expectedHours int
	}{
		{
			name:          "Empty stamps",
			timestamps:    []int64{},
			sessionWindow: 60,
			expectedHours: 0,
		},
		{
			name:          "Single commit",
			timestamps:    []int64{100000},
			sessionWindow: 60,
			expectedHours: 1, // Minimum session half is 30m, rounded up to 1h
		},
		{
			name: "Two commits within window",
			timestamps: []int64{
				100000,
				100000 + 1800, // +30 mins
			},
			sessionWindow: 60,
			expectedHours: 1, // 30m init + 30m gap = 60m = 1 hour
		},
		{
			name: "Two commits outside window",
			timestamps: []int64{
				100000,
				100000 + 7200, // +2 hours
			},
			sessionWindow: 60,
			expectedHours: 1, // 2 separate sessions of 30m each = 60m = 1 hour
		},
		{
			name: "Long continuous session",
			timestamps: []int64{
				100000,
				100000 + 1800, // +30m
				100000 + 3600, // +60m total (+30m from prev)
				100000 + 5400, // +90m total (+30m from prev)
			},
			sessionWindow: 60,
			expectedHours: 2, // 30m init + 90m gaps = 120m = 2 hours
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hours := CalculateHours(tt.timestamps, tt.sessionWindow)
			if hours != tt.expectedHours {
				t.Errorf("expected %d hours, got %d", tt.expectedHours, hours)
			}
		})
	}
}

func TestCalculateMonths(t *testing.T) {
	// Let's use specific dates
	t1 := time.Date(2023, 1, 15, 12, 0, 0, 0, time.UTC).Unix()
	t2 := time.Date(2023, 1, 25, 12, 0, 0, 0, time.UTC).Unix()
	t3 := time.Date(2023, 2, 10, 12, 0, 0, 0, time.UTC).Unix()
	t4 := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC).Unix()

	tests := []struct {
		name           string
		timestamps     []int64
		expectedMonths int
	}{
		{
			name:           "Empty stamps",
			timestamps:     []int64{},
			expectedMonths: 0,
		},
		{
			name:           "Same month",
			timestamps:     []int64{t1, t2},
			expectedMonths: 1,
		},
		{
			name:           "Different months same year",
			timestamps:     []int64{t1, t3},
			expectedMonths: 2,
		},
		{
			name:           "Same month different year",
			timestamps:     []int64{t1, t4},
			expectedMonths: 2, // 2023-01 and 2024-01
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			months := CalculateMonths(tt.timestamps)
			if months != tt.expectedMonths {
				t.Errorf("expected %d months, got %d", tt.expectedMonths, months)
			}
		})
	}
}

func TestCalculateMaxGap(t *testing.T) {
	tests := []struct {
		name        string
		timestamps  []int64
		expectedGap int
	}{
		{
			name:        "Empty",
			timestamps:  []int64{},
			expectedGap: 0,
		},
		{
			name:        "Single",
			timestamps:  []int64{100000},
			expectedGap: 0,
		},
		{
			name: "Two commits same day",
			timestamps: []int64{
				100000,
				100000 + 3600,
			},
			expectedGap: 0,
		},
		{
			name: "Two commits 2 days apart",
			timestamps: []int64{
				100000,
				100000 + (2 * 86400),
			},
			expectedGap: 2,
		},
		{
			name: "Multiple commits with gaps",
			timestamps: []int64{
				100000,
				100000 + 86400,       // 1 day gap
				100000 + (5 * 86400), // 4 day gap from prev
				100000 + (6 * 86400), // 1 day gap from prev
			},
			expectedGap: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateMaxGap(tt.timestamps)
			if got != tt.expectedGap {
				t.Errorf("CalculateMaxGap() = %v, want %v", got, tt.expectedGap)
			}
		})
	}
}
