package service

import (
	"math"
	"testing"
)

func TestResolveLuckCheckinBetAmount(t *testing.T) {
	largeBet := 81058106151016.73
	oneUnitBelow := math.Nextafter(largeBet, math.Inf(-1))

	tests := []struct {
		name    string
		bet     float64
		balance float64
		useMax  bool
		want    float64
		ok      bool
	}{
		{name: "normal amount", bet: 2.5, balance: 10, want: 2.5, ok: true},
		{name: "exact balance", bet: 10, balance: 10, want: 10, ok: true},
		{name: "stale max uses locked current balance", bet: 10, balance: 8.5, useMax: true, want: 8.5, ok: true},
		{name: "max does not include balance received after confirmation", bet: 8.5, balance: 10, useMax: true, want: 8.5, ok: true},
		{name: "manual amount remains strict at large values", bet: largeBet, balance: oneUnitBelow, ok: false},
		{name: "zero bet", bet: 0, balance: 10, ok: false},
		{name: "non-finite bet", bet: math.Inf(1), balance: 10, ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := resolveLuckCheckinBetAmount(tt.bet, tt.balance, tt.useMax)
			if ok != tt.ok || got != tt.want {
				t.Fatalf("resolveLuckCheckinBetAmount(%v, %v, %v) = (%v, %v), want (%v, %v)", tt.bet, tt.balance, tt.useMax, got, ok, tt.want, tt.ok)
			}
		})
	}
}
