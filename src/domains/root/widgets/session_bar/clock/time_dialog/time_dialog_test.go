package time_dialog

import (
	"testing"
	"time"
)

func TestEnteredReadsMinutesThenSeconds(t *testing.T) {
	for _, tc := range []struct {
		digits []int
		want   time.Duration
	}{
		{[]int{5}, 50 * time.Minute},
		{[]int{0, 5}, 5 * time.Minute},
		{[]int{0, 1, 3}, time.Minute + 30*time.Second},
		{[]int{1, 2, 3, 4}, 12*time.Minute + 34*time.Second},
	} {
		if got := entered(tc.digits); got != tc.want {
			t.Errorf("entered(%v) = %v, want %v", tc.digits, got, tc.want)
		}
	}
}
