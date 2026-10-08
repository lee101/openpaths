package handler

import "testing"

func TestRelightBillableSeconds(t *testing.T) {
	for _, tc := range []struct{ in, want float64 }{{1.2, 3}, {2.33, 3}, {3, 3}, {3.01, 4}, {9.5, 10}, {15, 15}, {17, 15}} {
		if got := relightBillableSeconds(tc.in); got != tc.want {
			t.Errorf("%v: got %v want %v", tc.in, got, tc.want)
		}
	}
}
