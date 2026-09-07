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
// with ErrExponentOutOfRange if the resulting scale (d.Scale() * n) would
// exceed maxParseDigits, the same bound Parse uses to reject inputs that
// would otherwise force a multi-gigabyte allocation.
func (d Decimal) Pow(n uint) (Decimal, error) {
	if n == 0 {
		return NewFromInt64(1), nil
	}

	scale := int64(d.scale) * int64(n)
	if scale > maxParseDigits {
		return Decimal{}, ErrExponentOutOfRange
	}

	coeff := new(big.Int).Exp(d.coefficient(), new(big.Int).SetUint64(uint64(n)), nil)
	return Decimal{coeff: coeff, scale: int32(scale)}, nil
}
