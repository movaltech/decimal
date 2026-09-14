package decimal_test

import (
	"errors"
	"testing"

	"github.com/movaltech/decimal"
)

func TestMoneyMul(t *testing.T) {
	price, _ := decimal.ParseMoney("19.99", "USD", decimal.RoundHalfEven)
	got := price.Mul(decimal.NewFromInt64(3))
	want, _ := decimal.ParseMoney("59.97", "USD", decimal.RoundHalfEven)
	if eq, err := got.Cmp(want); err != nil || eq != 0 {
		t.Fatalf("price.Mul(3) = %s, want %s", got, want)
	}
	if got.Currency != "USD" {
		t.Fatalf("price.Mul(3) currency = %q, want USD", got.Currency)
	}
}

func TestMoneyCmp(t *testing.T) {
	a, _ := decimal.ParseMoney("10.00", "USD", decimal.RoundHalfEven)
	b, _ := decimal.ParseMoney("20.00", "USD", decimal.RoundHalfEven)

	if got, err := a.Cmp(b); err != nil || got != -1 {
		t.Fatalf("a.Cmp(b) = %d, %v; want -1, nil", got, err)
	}
	if got, err := b.Cmp(a); err != nil || got != 1 {
		t.Fatalf("b.Cmp(a) = %d, %v; want 1, nil", got, err)
	}
	if got, err := a.Cmp(a); err != nil || got != 0 {
		t.Fatalf("a.Cmp(a) = %d, %v; want 0, nil", got, err)
	}

	eur, _ := decimal.ParseMoney("10.00", "EUR", decimal.RoundHalfEven)
	if _, err := a.Cmp(eur); !errors.Is(err, decimal.ErrCurrencyMismatch) {
		t.Fatalf("a.Cmp(eur) err = %v, want ErrCurrencyMismatch", err)
	}
}

func TestMoneyNeg(t *testing.T) {
	a, _ := decimal.ParseMoney("10.00", "USD", decimal.RoundHalfEven)
	neg := a.Neg()
	if neg.Currency != "USD" {
		t.Fatalf("Neg() currency = %q, want USD", neg.Currency)
	}
	if !neg.Amount.Equal(decimal.MustParse("-10.00")) {
		t.Fatalf("Neg() amount = %s, want -10.00", neg.Amount)
	}
	if !neg.Neg().Amount.Equal(a.Amount) {
		t.Fatalf("Neg().Neg() should round-trip back to the original amount")
	}
}

func TestMoneyIsZero(t *testing.T) {
	zero, _ := decimal.ParseMoney("0.00", "USD", decimal.RoundHalfEven)
	nonZero, _ := decimal.ParseMoney("0.01", "USD", decimal.RoundHalfEven)

	if !zero.IsZero() {
		t.Fatalf("zero.IsZero() = false, want true")
	}
	if nonZero.IsZero() {
		t.Fatalf("nonZero.IsZero() = true, want false")
	}
}
