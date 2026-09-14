package decimal_test

import (
	"errors"
	"testing"

	"github.com/movaltech/decimal"
)

func TestDecimalPow(t *testing.T) {
	cases := []struct {
		base string
		n    uint
		want string
	}{
		{"1.05", 0, "1"},
		{"1.05", 1, "1.05"},
		{"1.05", 2, "1.1025"},
		{"2", 10, "1024"},
		{"0", 5, "0"},
		{"0", 0, "1"}, // conventional: x^0 == 1, even for x == 0
	}
	for _, c := range cases {
		got, err := decimal.MustParse(c.base).Pow(c.n)
		if err != nil {
			t.Fatalf("MustParse(%q).Pow(%d) unexpected error: %v", c.base, c.n, err)
		}
		if !got.Equal(decimal.MustParse(c.want)) {
			t.Errorf("MustParse(%q).Pow(%d) = %s, want %s", c.base, c.n, got, c.want)
		}
	}
}

func TestDecimalPowExponentOutOfRange(t *testing.T) {
	// scale(1) * n(1_000_001) exceeds the internal maxParseDigits bound, and
	// must be rejected before attempting the (otherwise enormous) big.Int.Exp.
	_, err := decimal.MustParse("1.0").Pow(1_000_001)
	if !errors.Is(err, decimal.ErrExponentOutOfRange) {
		t.Fatalf("Pow(1_000_001) err = %v, want ErrExponentOutOfRange", err)
	}
}

func TestDecimalPowIntegerBaseCoefficientOutOfRange(t *testing.T) {
	// d.Scale() is 0 here, so scale(0) * n is always 0 and the scale check
	// alone would never reject this — it's the coefficient (2^n has roughly
	// n*log10(2) digits) that must be bounded instead. Before this guard
	// existed, this call would run for an unreasonable amount of time and
	// memory instead of failing immediately.
	_, err := decimal.NewFromInt64(2).Pow(4_000_000_000)
	if !errors.Is(err, decimal.ErrExponentOutOfRange) {
		t.Fatalf("Pow(4_000_000_000) err = %v, want ErrExponentOutOfRange", err)
	}
}
