package decimal

import "math/big"

// NewFromMinorUnits builds a Decimal from an integer count of minor currency
// units (e.g. cents) and the number of minor-unit digits for that currency
// (2 for USD/EUR, 0 for JPY, 3 for BHD). This is the recommended way to move
// money in and out of systems (payment processors, ledgers) that represent
// amounts as integers to sidestep floating point entirely.
//
//	decimal.NewFromMinorUnits(1050, 2) // 10.50 USD
//	decimal.NewFromMinorUnits(500, 0)  // 500 JPY (no minor unit)
func NewFromMinorUnits(minorUnits int64, minorDigits int32) Decimal {
	return New(minorUnits, minorDigits)
}

// MinorUnits returns d as an integer count of minor units at minorDigits
// precision (rounding if necessary), and an error if RoundUnnecessary is
// requested and d does not divide evenly. This is the inverse of
// NewFromMinorUnits and is typically the last step before handing an amount
// to a system that only accepts integers.
func (d Decimal) MinorUnits(minorDigits int32, mode RoundingMode) (int64, error) {
	r, err := d.Rescale(minorDigits, mode)
	if err != nil {
		return 0, err
	}
	scaled := r.rescaleUpTo(minorDigits) // no-op if already at minorDigits, exact if not
	if !scaled.coefficient().IsInt64() {
		return 0, ErrPrecisionOutOfRange
	}
	return scaled.coefficient().Int64(), nil
}

// Int64 returns d truncated to an integer (RoundDown) and whether the
// conversion was exact (i.e. d had no fractional part and fits in an int64).
// Prefer Rescale/MinorUnits with an explicit rounding mode for money; Int64
// is a convenience for cases where d is already known to be a whole number.
func (d Decimal) Int64() (value int64, exact bool) {
	r, _ := d.Rescale(0, RoundDown)
	return r.coefficient().Int64(), d.Cmp(r) == 0 && r.coefficient().IsInt64()
}

// BigInt returns the unscaled coefficient of d as a *big.Int, i.e. the
// integer such that BigInt() * 10^-Scale() == d. The returned value is a
// copy and safe to mutate.
func (d Decimal) BigInt() *big.Int {
	return new(big.Int).Set(d.coefficient())
}

// Float64 converts d to the nearest float64. This conversion is inherently
// lossy for the same reason Decimal exists in the first place, so it exists
// only for interop with float-based APIs (plotting, non-monetary math) —
// never use it to move a value into or out of a money calculation.
func (d Decimal) Float64() (float64, bool) {
	num := new(big.Rat).SetInt(d.coefficient())
	den := new(big.Rat).SetInt(exp10(d.scale))
	num.Quo(num, den)
	f, exact := num.Float64()
	return f, exact
}
