package decimal

import "math/big"

// Pow returns d raised to the non-negative integer power n, exactly:
// (coeff * 10^-scale)^n == coeff^n * 10^-(scale*n), so the coefficient and
// scale are simply exponentiated/multiplied — no rounding is ever involved,
// which is what makes this safe to use for compound interest and annuity
// calculations (e.g. (1 + rate).Pow(periods)) without accumulating error
// across periods.
//
// Pow(0) is 1 for any d, including 0 (the conventional definition). It fails
// with ErrExponentOutOfRange if the resulting scale (d.Scale() * n) or the
// resulting coefficient's digit count (roughly digits(d) * n) would exceed
// maxParseDigits, the same bound Parse uses to reject inputs that would
// otherwise force a multi-gigabyte allocation. The coefficient check matters
// even when d.Scale() is 0 — an integer base raised to a large enough power
// (e.g. NewFromInt64(2).Pow(4_000_000_000)) grows the same way regardless of
// scale, so the scale check alone would not catch it.
func (d Decimal) Pow(n uint) (Decimal, error) {
	if n == 0 {
		return NewFromInt64(1), nil
	}

	scale := int64(d.scale) * int64(n)
	if scale > maxParseDigits {
		return Decimal{}, ErrExponentOutOfRange
	}

	coeff := d.coefficient()
	if coeff.Sign() != 0 && n > 1 {
		// digits(coeff) <= coeff.BitLen()/3 + 1, since log2(10) > 3; cheap to
		// compute via BitLen even for a large coeff, unlike a base-10
		// conversion, and avoids ever calling Exp with an exponent large
		// enough to make the result itself the size of the bomb.
		digitBound := int64(coeff.BitLen())/3 + 1
		if digitBound > maxParseDigits/int64(n) {
			return Decimal{}, ErrExponentOutOfRange
		}
	}

	result := new(big.Int).Exp(coeff, new(big.Int).SetUint64(uint64(n)), nil)
	return Decimal{coeff: result, scale: int32(scale)}, nil
}
