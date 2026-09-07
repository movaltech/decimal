package decimal

import (
	"math/big"
	"math/rand"
	"testing"
)

// TestQuoScaleAgainstBigRat is the real correctness proof: for many random
// (numerator, denominator, scale) triples, the HalfEven result of QuoScale
// must equal what you get by computing the exact rational a/b with
// math/big.Rat and rounding half-to-even by hand. This checks the rounding
// logic against ground truth instead of a handful of hand-picked examples.
func TestQuoScaleAgainstBigRat(t *testing.T) {
	rng := rand.New(rand.NewSource(42))

	for i := 0; i < 5000; i++ {
		aUnscaled := rng.Int63n(2_000_000) - 1_000_000
		bUnscaled := rng.Int63n(2_000_000) - 1_000_000
		if bUnscaled == 0 {
			continue
		}
		aScale := int32(rng.Intn(4))
		bScale := int32(rng.Intn(4))
		scale := int32(rng.Intn(6))

		a := New(aUnscaled, aScale)
		b := New(bUnscaled, bScale)

		got, err := a.QuoScale(b, scale, RoundHalfEven)
		if err != nil {
			t.Fatalf("QuoScale(%s,%s,scale=%d): %v", a, b, scale, err)
		}

		want := exactHalfEven(t, aUnscaled, aScale, bUnscaled, bScale, scale)
		if got.Cmp(want) != 0 {
			t.Fatalf("a=%s b=%s scale=%d: got %s, want %s", a, b, scale, got, want)
		}
	}
}

// exactHalfEven computes a/b rounded to `scale` digits using math/big.Rat as
// an independent ground truth, implementing half-to-even by hand on the
// exact rational remainder.
func exactHalfEven(t *testing.T, aUnscaled int64, aScale int32, bUnscaled int64, bScale int32, scale int32) Decimal {
	t.Helper()
	a := new(big.Rat).SetFrac(big.NewInt(aUnscaled), pow10Rat(aScale))
	b := new(big.Rat).SetFrac(big.NewInt(bUnscaled), pow10Rat(bScale))
	q := new(big.Rat).Quo(a, b)

	// q * 10^scale, split into integer part and remainder fraction.
	scaled := new(big.Rat).Mul(q, new(big.Rat).SetInt(exp10(scale)))
	neg := scaled.Sign() < 0
	absScaled := new(big.Rat).Abs(scaled)

	num := absScaled.Num()
	den := absScaled.Denom()
	quo, rem := new(big.Int).QuoRem(num, den, new(big.Int))

	twiceRem := new(big.Int).Lsh(rem, 1)
	switch twiceRem.Cmp(den) {
	case 1:
		quo.Add(quo, big.NewInt(1))
	case 0:
		if new(big.Int).And(quo, big.NewInt(1)).Sign() != 0 {
			quo.Add(quo, big.NewInt(1))
		}
	}
	if neg {
		quo.Neg(quo)
	}
	return Decimal{coeff: quo, scale: scale}
}

func pow10Rat(n int32) *big.Int {
	return new(big.Int).Set(exp10(n))
}

func TestNegScaleAndBigIntAccessors(t *testing.T) {
	d := MustParse("-5.00")
	if d.Neg().String() != "5.00" {
		t.Fatal("Neg failed")
	}
	if d.Scale() != 2 {
		t.Fatalf("Scale() = %d, want 2", d.Scale())
	}
	if d.BigInt().Int64() != -500 {
		t.Fatalf("BigInt() = %v, want -500", d.BigInt())
	}
}

func TestInt64Conversion(t *testing.T) {
	v, exact := MustParse("42").Int64()
	if !exact || v != 42 {
		t.Fatalf("got (%d,%v), want (42,true)", v, exact)
	}
	v2, exact2 := MustParse("42.50").Int64()
	if exact2 || v2 != 42 {
		t.Fatalf("got (%d,%v), want (42,false)", v2, exact2)
	}
}

func TestFloat64Interop(t *testing.T) {
	f, _ := MustParse("2.5").Float64()
	if f != 2.5 {
		t.Fatalf("got %v, want 2.5", f)
	}
}

func TestNewFromBigInt(t *testing.T) {
	src := big.NewInt(1234)
	d := NewFromBigInt(src, 2)
	src.SetInt64(0) // mutate the caller's copy; d must be unaffected
	if d.String() != "12.34" {
		t.Fatalf("got %s, want 12.34 (NewFromBigInt must copy)", d)
	}
}

func TestSQLValueAndScan(t *testing.T) {
	d := MustParse("19.99")
	v, err := d.Value()
	if err != nil || v != "19.99" {
		t.Fatalf("Value() = (%v,%v), want (19.99,nil)", v, err)
	}

	var out Decimal
	for _, src := range []interface{}{"19.99", []byte("19.99"), int64(19)} {
		if err := out.Scan(src); err != nil {
			t.Fatalf("Scan(%v): %v", src, err)
		}
	}
	if err := out.Scan(3.14); err != nil {
		t.Fatalf("Scan(float64): %v", err)
	}
	if err := out.Scan(nil); err != nil || !out.IsZero() {
		t.Fatalf("Scan(nil) should reset to zero, got %s, err=%v", out, err)
	}
}

func TestMoneySubAndNegativeAllocation(t *testing.T) {
	a, _ := ParseMoney("5.00", "USD", RoundHalfEven)
	b, _ := ParseMoney("2.00", "USD", RoundHalfEven)
	diff, err := a.Sub(b)
	if err != nil || diff.Amount.String() != "3.00" {
		t.Fatalf("got %v %v, want 3.00", diff, err)
	}

	neg, _ := ParseMoney("-100.00", "USD", RoundHalfEven)
	shares, err := neg.AllocateEven(3)
	if err != nil {
		t.Fatal(err)
	}
	sum := shares[0].Amount.Add(shares[1].Amount).Add(shares[2].Amount)
	if !sum.Equal(neg.Amount) {
		t.Fatalf("negative allocation sums to %s, want %s", sum, neg.Amount)
	}
}

func TestRoundingModeString(t *testing.T) {
	if RoundHalfEven.String() != "HalfEven" {
		t.Fatalf("got %q", RoundHalfEven.String())
	}
}

func TestMoneyString(t *testing.T) {
	m, _ := ParseMoney("10.50", "USD", RoundHalfEven)
	if m.String() != "10.50 USD" {
		t.Fatalf("got %q", m.String())
	}
}
