package decimal

import "math/big"

// Add returns d + e, exactly. Addition never loses precision: operands are
// aligned to the larger of the two scales (which only ever pads with exact
// trailing zeros) before the coefficients are summed.
func (d Decimal) Add(e Decimal) Decimal {
	scale := maxInt32(d.scale, e.scale)
	a := d.rescaleUpTo(scale)
	b := e.rescaleUpTo(scale)
	return Decimal{coeff: new(big.Int).Add(a.coefficient(), b.coefficient()), scale: scale}
}

// Sub returns d - e, exactly, by the same reasoning as Add.
func (d Decimal) Sub(e Decimal) Decimal {
	scale := maxInt32(d.scale, e.scale)
	a := d.rescaleUpTo(scale)
	b := e.rescaleUpTo(scale)
	return Decimal{coeff: new(big.Int).Sub(a.coefficient(), b.coefficient()), scale: scale}
}

// Mul returns d * e, exactly. Multiplication of fixed-point values is always
// exact in this representation (coefficients multiply, scales add), so Mul
// takes no rounding mode and cannot fail.
func (d Decimal) Mul(e Decimal) Decimal {
	return Decimal{
		coeff: new(big.Int).Mul(d.coefficient(), e.coefficient()),
		scale: d.scale + e.scale,
	}
}

// Rescale returns d expressed at exactly newScale digits after the decimal
// point. Increasing the scale is always exact. Decreasing the scale rounds
// using mode, and returns ErrRoundingNecessary if mode is RoundUnnecessary
// and the discarded digits are non-zero. newScale must be >= 0.
//
// This is the operation behind fixing a value to a currency's minor unit,
// e.g. rescaling a computed tax amount to 2 places before storing it as
// cents: total.Rescale(2, decimal.RoundHalfEven).
func (d Decimal) Rescale(newScale int32, mode RoundingMode) (Decimal, error) {
	if newScale < 0 || newScale > maxParseDigits {
		return Decimal{}, ErrScaleOutOfRange
	}
	if newScale >= d.scale {
		return d.rescaleUpTo(newScale), nil
	}

	diff := d.scale - newScale
	divisor := exp10(diff)
	coeff := d.coefficient()
	neg := coeff.Sign() < 0

	abs := new(big.Int).Abs(coeff)
	q, r := new(big.Int).QuoRem(abs, divisor, new(big.Int))
	q, err := applyRounding(q, r, divisor, neg, mode)
	if err != nil {
		return Decimal{}, err
	}
	if neg && q.Sign() != 0 {
		q.Neg(q)
	}
	return Decimal{coeff: q, scale: newScale}, nil
}

// QuoScale divides d by e and returns a result truncated/rounded to exactly
// scale digits after the decimal point, using mode to resolve the remainder.
// This is the natural money-division primitive: you always state up front
// how many decimal places the answer must have, so a non-terminating
// quotient (e.g. splitting $100 three ways) can never silently produce a
// runaway-precision or truncated-without-warning result.
func (d Decimal) QuoScale(e Decimal, scale int32, mode RoundingMode) (Decimal, error) {
	if e.IsZero() {
		return Decimal{}, ErrDivisionByZero
	}
	if scale < 0 || scale > maxParseDigits {
		return Decimal{}, ErrScaleOutOfRange
	}

	// result_coeff / 10^scale ≈ (d.coeff / 10^d.scale) / (e.coeff / 10^e.scale)
	// => result_coeff = d.coeff * 10^(scale + e.scale) / (e.coeff * 10^d.scale)
	numExp := int64(scale) + int64(e.scale)
	numerator := new(big.Int).Mul(d.coefficient(), exp10FromInt64(numExp))
	denominator := new(big.Int).Mul(e.coefficient(), exp10(d.scale))

	neg := (numerator.Sign() < 0) != (denominator.Sign() < 0)
	numerator.Abs(numerator)
	denominator.Abs(denominator)

	q, r := new(big.Int).QuoRem(numerator, denominator, new(big.Int))
	q, err := applyRounding(q, r, denominator, neg, mode)
	if err != nil {
		return Decimal{}, err
	}
	if neg && q.Sign() != 0 {
		q.Neg(q)
	}
	return Decimal{coeff: q, scale: scale}, nil
}

// MaxExactScale bounds how many fractional digits QuoExact will search
// through before giving up on a non-terminating division.
const MaxExactScale = 100

// QuoExact divides d by e and returns the mathematically exact result,
// failing with ErrInexact if the quotient does not terminate within
// MaxExactScale digits (e.g. 1/3 = 0.333... never terminates). Use this only
// when you specifically need to detect and reject inexact division, such as
// validating that a per-unit price divides a total evenly; for money
// calculations that must always produce an answer, use QuoScale instead.
func (d Decimal) QuoExact(e Decimal) (Decimal, error) {
	if e.IsZero() {
		return Decimal{}, ErrDivisionByZero
	}
	if d.IsZero() {
		return Decimal{}, nil
	}
	// e is non-zero (checked above) and scale only ever ranges over
	// [0, MaxExactScale], so the only way QuoScale can fail here is the
	// RoundUnnecessary remainder check; any other error would mean the
	// invariants above no longer hold.
	for scale := int32(0); scale <= MaxExactScale; scale++ {
		result, err := d.QuoScale(e, scale, RoundUnnecessary)
		if err == nil {
			return result, nil
		}
	}
	return Decimal{}, ErrInexact
}

// exp10FromInt64 is exp10 for a 64-bit exponent. n is always >= 0 here: it
// is scale+e.scale from QuoScale, and both operands are always >= 0.
func exp10FromInt64(n int64) *big.Int {
	if n < int64(len(powersOf10)) {
		return powersOf10[n]
	}
	return new(big.Int).Exp(bigTen, big.NewInt(n), nil)
}
