package handler

import "testing"

func TestFormatUSDExact(t *testing.T) {
	tests := []struct {
		name   string
		units  int64
		expect string
	}{
		{name: "zero", units: 0, expect: "0.0000"},
		{name: "whole dollars", units: 100000, expect: "10.0000"},
		{name: "sub-cent balance", units: 99999, expect: "9.9999"},
		{name: "negative amount", units: -1, expect: "-0.0001"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatUSDExact(tt.units); got != tt.expect {
				t.Fatalf("formatUSDExact(%d) = %q, want %q", tt.units, got, tt.expect)
			}
		})
	}
}
