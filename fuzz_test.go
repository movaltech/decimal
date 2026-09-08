package decimal

import "testing"

// FuzzParse checks that Parse never panics and never hangs on arbitrary
// input, and that whenever it succeeds, the result round-trips through
// String()+Parse to the same value. Parse is the package's documented entry
// point for untrusted external text (user input, JSON, CSV), so it is the
// one function in this package that must survive adversarial fuzzing, not
// just hand-picked test cases.
//
// Run with: go test -fuzz=FuzzParse -fuzztime=60s
func FuzzParse(f *testing.F) {
	seeds := []string{
		"0", "1", "-1", "0.1", "-0.1", "1.5e2", "150e-2", "1e999999999",
		"", "-", ".", "1.2.3", "abc", "1e", "1.2e", "+5", "5e0", "0.000",
		"999999999999999999999999999999", "1e-999999999", "1e-2000000",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, s string) {
		d, err := Parse(s)
		if err != nil {
			return
		}
		back, err := Parse(d.String())
		if err != nil {
			t.Fatalf("Parse(%q) succeeded as %s, but Parse(String()) failed: %v", s, d, err)
		}
		if !back.Equal(d) {
			t.Fatalf("round-trip mismatch: Parse(%q) = %s, Parse(that.String()=%q) = %s", s, d, d.String(), back)
		}
	})
}

// FuzzRescale checks that Rescale never panics for any coefficient/scale
// combination reachable via Parse, across every rounding mode.
func FuzzRescale(f *testing.F) {
	f.Add("123.456", int32(2), uint8(RoundHalfEven))
	f.Add("0", int32(0), uint8(RoundUp))
	f.Add("-999.999", int32(0), uint8(RoundFloor))

	f.Fuzz(func(t *testing.T, s string, newScale int32, mode uint8) {
		d, err := Parse(s)
		if err != nil {
			return
		}
		if mode > uint8(RoundUnnecessary) {
			return
		}
		_, _ = d.Rescale(newScale, RoundingMode(mode))
	})
}

// FuzzQuoScale checks that QuoScale never panics or hangs for any
// dividend/divisor/scale/mode combination — including an oversized scale,
// which must be rejected by the ErrScaleOutOfRange guard rather than
// attempting to materialize the result — and that whenever it succeeds, the
// result actually has the scale that was requested, per its documented
// contract.
//
// Run with: go test -fuzz=FuzzQuoScale -fuzztime=60s
func FuzzQuoScale(f *testing.F) {
	f.Add("1", "3", int32(2), uint8(RoundHalfUp))
	f.Add("100", "0", int32(2), uint8(RoundHalfEven))
	f.Add("1", "3", int32(2_000_000_000), uint8(RoundDown))
	f.Add("-7.5", "2.5", int32(4), uint8(RoundCeiling))
	f.Add("0", "1", int32(0), uint8(RoundUnnecessary))

	f.Fuzz(func(t *testing.T, dividend, divisor string, scale int32, mode uint8) {
		if mode > uint8(RoundUnnecessary) {
			return
		}
		d, err := Parse(dividend)
		if err != nil {
			return
		}
		e, err := Parse(divisor)
		if err != nil {
			return
		}
		result, err := d.QuoScale(e, scale, RoundingMode(mode))
		if err != nil {
			return
		}
		if result.Scale() != scale {
			t.Fatalf("%s.QuoScale(%s, %d, %s) = %s, want scale %d", d, e, scale, RoundingMode(mode), result, scale)
		}
	})
}

// FuzzPow checks that Pow never panics or hangs for any base/exponent
// combination — including an integer base raised to a huge exponent, which
// must be rejected by the coefficient-size guard rather than computed —
// and that for small exponents (cheap to verify independently), the result
// matches repeated Mul exactly.
//
// Run with: go test -fuzz=FuzzPow -fuzztime=60s
func FuzzPow(f *testing.F) {
	f.Add("1.05", uint(12))
	f.Add("0", uint(5))
	f.Add("0", uint(0))
	f.Add("2", uint(4_000_000_000))
	f.Add("-3.5", uint(3))

	f.Fuzz(func(t *testing.T, base string, n uint) {
		d, err := Parse(base)
		if err != nil {
			return
		}
		result, err := d.Pow(n)
		if err != nil {
			return
		}
		if n <= 20 {
			want := NewFromInt64(1)
			for i := uint(0); i < n; i++ {
				want = want.Mul(d)
			}
			if !result.Equal(want) {
				t.Fatalf("%s.Pow(%d) = %s, want %s", d, n, result, want)
			}
		}
	})
}

// FuzzMoney checks that ParseMoney and every Money method never panic on
// arbitrary amount/currency input, and that AllocateEven's shares always
// sum back to exactly the original amount — the one property the method
// exists to guarantee.
//
// Run with: go test -fuzz=FuzzMoney -fuzztime=60s
func FuzzMoney(f *testing.F) {
	f.Add("100.00", "USD", "50.00", "EUR", uint8(RoundHalfEven), 3)
	f.Add("0.00", "JPY", "0.00", "JPY", uint8(RoundHalfEven), 1)
	f.Add("-10.005", "BHD", "10.005", "BHD", uint8(RoundHalfUp), 7)
	f.Add("99999999999999999999.99", "USD", "1.00", "USD", uint8(RoundDown), 5)
	f.Add("10.00", "USD", "10.00", "USD", uint8(RoundHalfEven), 0)

	f.Fuzz(func(t *testing.T, amount1, currency1, amount2, currency2 string, mode uint8, n int) {
		if mode > uint8(RoundUnnecessary) {
			return
		}
		m1, err := ParseMoney(amount1, currency1, RoundingMode(mode))
		if err != nil {
			return
		}
		m2, err := ParseMoney(amount2, currency2, RoundingMode(mode))
		if err != nil {
			return
		}

		if m1.Amount.Scale() != MinorDigitsFor(currency1) {
			t.Fatalf("ParseMoney(%q, %q) amount scale = %d, want %d", amount1, currency1, m1.Amount.Scale(), MinorDigitsFor(currency1))
		}

		_, _ = m1.Add(m2)
		_, _ = m1.Sub(m2)
		_, _ = m1.Cmp(m2)
		_ = m1.Neg()
		_ = m1.IsZero()
		_ = m1.String()

		scaled := m1.Mul(m2.Amount)
		if scaled.Currency != m1.Currency {
			t.Fatalf("Mul changed currency: got %s, want %s", scaled.Currency, m1.Currency)
		}
		if !scaled.Amount.Equal(m1.Amount.Mul(m2.Amount)) {
			t.Fatalf("%s.Mul(%s) = %s, want %s", m1, m2.Amount, scaled, m1.Amount.Mul(m2.Amount))
		}

		// Bound the share count so each fuzz iteration stays fast; the
		// interesting cases (zero, negative, and a handful of shares) are
		// all small.
		shareCount := n % 20
		if shareCount < 0 {
			shareCount = -shareCount
		}
		shares, err := m1.AllocateEven(shareCount)
		if shareCount <= 0 {
			if err == nil {
				t.Fatalf("AllocateEven(%d) succeeded, want error", shareCount)
			}
			return
		}
		if err != nil {
			// e.g. ErrPrecisionOutOfRange when the amount doesn't fit an
			// int64 count of minor units — a legitimate rejection, not a bug.
			return
		}
		sum := Money{Amount: Zero(), Currency: m1.Currency}
		for _, s := range shares {
			sum, err = sum.Add(s)
			if err != nil {
				t.Fatalf("summing shares of %s: %v", m1, err)
			}
		}
		if sum.Amount.Cmp(m1.Amount) != 0 {
			t.Fatalf("%s.AllocateEven(%d) shares %v sum to %s, want %s", m1, shareCount, shares, sum, m1)
		}
	})
}
