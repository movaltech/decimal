package decimal

import "math/big"

// Context bundles a significant-digit precision and a rounding mode. Where
// QuoScale and Rescale round to a fixed number of digits *after the point*
// (what you want for money, which has a fixed number of minor-unit digits),
// a Context rounds to a fixed number of *significant* digits (what you want
// for general numeric computation, e.g. interest-rate or scientific
// factors).
type Context struct {
	// Precision is the maximum number of significant digits a result may
	// have. Zero means unlimited precision (only explicit Rescale/QuoScale
	// calls ever round).
	Precision uint32
	Rounding  RoundingMode
}

// DefaultContext is a reasonable general-purpose default: 28 significant
// digits with banker's rounding.
var DefaultContext = Context{Precision: 28, Rounding: RoundHalfEven}

// Round returns d rounded to c.Precision significant digits using
// c.Rounding. If d already fits within the precision, or the context is
// unlimited (Precision == 0), d is returned unchanged.
//
// Unlike Rescale, dropped digits are not limited to the fractional part: if
// the integer part alone has more digits than c.Precision (e.g. 123456 at
// Precision 3), the integer part is rounded too and re-padded with trailing
// zeros to preserve magnitude (123456 -> 123000).
func (c Context) Round(d Decimal) (Decimal, error) {
	if c.Precision == 0 {
		return d, nil
	}
	digits := numDigits(d.coefficient())
	if digits <= int(c.Precision) {
		return d, nil
	}

	drop := digits - int(c.Precision)
	divisor := exp10(int32(drop))
	coeff := d.coefficient()
	neg := coeff.Sign() < 0

	abs := new(big.Int).Abs(coeff)
	q, r := new(big.Int).QuoRem(abs, divisor, new(big.Int))
	q, err := applyRounding(q, r, divisor, neg, c.Rounding)
	if err != nil {
		return Decimal{}, err
	}

	extra := int32(drop)
	// Rounding up an all-nines quotient carries into one extra digit (e.g.
	// 999 -> 1000); that carry digit is always an exact trailing zero, so
	// shifting it back out keeps the result at exactly c.Precision digits.
	if numDigits(q) > int(c.Precision) {
		q = new(big.Int).Quo(q, bigTen)
		extra++
	}
	if neg && q.Sign() != 0 {
		q.Neg(q)
	}

	newScale := d.scale - extra
	if newScale >= 0 {
		return Decimal{coeff: q, scale: newScale}, nil
	}
	// extra digits were dropped from the integer part itself: re-pad with
	// zeros to restore magnitude, at scale 0 (scale can never be negative).
	pad := exp10(-newScale)
	return Decimal{coeff: new(big.Int).Mul(q, pad), scale: 0}, nil
}

// QuoContext divides d by e and rounds the result to c.Precision significant
// digits using c.Rounding — the significant-digit counterpart to QuoScale's
// fixed decimal places.
//
// Because most divisions do not terminate, the quotient is first computed
// with generous guard digits and truncated, then rounded to the requested
// precision, which is how arbitrary-precision decimal division avoids
// running forever on an input like 1/3.
func (d Decimal) QuoContext(e Decimal, c Context) (Decimal, error) {
	if e.IsZero() {
		return Decimal{}, ErrDivisionByZero
	}
	prec := c.Precision
	if prec == 0 {
		prec = DefaultContext.Precision
	}
	const guardDigits = 10
	workingScale := maxInt32(d.scale, e.scale) + int32(prec) + guardDigits

	intermediate, err := d.QuoScale(e, workingScale, RoundDown)
	if err != nil {
		return Decimal{}, err
	}
	return Context{Precision: prec, Rounding: c.Rounding}.Round(intermediate)
}
