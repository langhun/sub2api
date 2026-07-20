package service

import (
	"math"
	"testing"
)

func TestNormalizeLuckCheckinBetAmount(t *testing.T) {
	largeBet := 81058106151016.73
	oneUnitBelow := math.Nextafter(largeBet, math.Inf(-1))
	twoUnitsAbove := math.Nextafter(largeBet, math.Inf(1))

	tests := []struct {
		name    string
		bet     float64
		balance float64
		want    float64
		ok      bool
	}{
		{name: "normal amount", bet: 2.5, balance: 10, want: 2.5, ok: true},
		{name: "exact balance", bet: 10, balance: 10, want: 10, ok: true},
		{name: "large max rounded by one unit", bet: largeBet, balance: oneUnitBelow, want: oneUnitBelow, ok: true},
		{name: "more than one unit over balance", bet: twoUnitsAbove, balance: oneUnitBelow, ok: false},
		{name: "zero bet", bet: 0, balance: 10, ok: false},
		{name: "non-finite bet", bet: math.Inf(1), balance: 10, ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := normalizeLuckCheckinBetAmount(tt.bet, tt.balance)
			if ok != tt.ok || got != tt.want {
				t.Fatalf("normalizeLuckCheckinBetAmount(%v, %v) = (%v, %v), want (%v, %v)", tt.bet, tt.balance, got, ok, tt.want, tt.ok)
			}
		})
	}
}
